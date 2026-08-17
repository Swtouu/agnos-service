# Agnos Hospital Middleware API

A backend middleware for searching and displaying patient information across hospitals, with per-hospital staff isolation. Built for the Agnos back-end developer candidate assignment.

## Tech stack

Go, Gin, GORM, PostgreSQL, `golang-migrate`, Docker, Nginx. Tests: standard `testing` package, plus `testcontainers-go` for the repository layer.

## Quick start

```bash
cp .env.example .env   # edit if you want non-default secrets
docker compose up --build
```

This starts, in order: Postgres → `seed` (self-migrates the schema, then populates fixture data, exits, skips seeding if already seeded) → `api` (self-migrates again — idempotent no-op since `seed` already did it — then serves) → `nginx` (reverse proxy on `http://localhost:8080`).

Health check: `curl http://localhost:8080/healthz`

Swagger UI: `http://localhost:8080/swagger/index.html`

### Test UI

`webui/index.html` is a minimal, single-file, no-build test console (vanilla HTML/JS) styled as a lightweight hospital staff portal — login, patient search with a results table, staff creation, refresh/logout, and a mock-HIS lookup, each hitting its real endpoint with a request/response log at the bottom. It's a testing aid, not the graded front-end deliverable (that's a separate assignment with its own Next.js/WebSocket spec).

```bash
cd webui && python3 -m http.server 5500
# open http://localhost:5500
```

The webui needs CORS enabled on the API — set `ENABLE_DEV_CORS=true` in `.env` (see `.env.example`) before `docker compose up`. Off by default; when on, the API allows CORS from any origin (`internal/middleware/cors.go`) — permissive by design, dev-only, gated behind the env var specifically so it can't ship on in a real deployment by accident.

## Seeded accounts

Two hospitals, four staff, nine patients (see `cmd/seed/main.go` for the full fixture set — includes intentionally overlapping patient names across hospitals to make tenant-isolation testing meaningful).

| hospital     | username    | password      |
|--------------|-------------|---------------|
| `hospital_a` | `staff_a1`  | `Password123!`|
| `hospital_a` | `staff_a2`  | `Password123!`|
| `hospital_b` | `staff_b1`  | `Password123!`|
| `hospital_b` | `staff_b2`  | `Password123!`|

Example flow:

```bash
TOKEN=$(curl -s -X POST localhost:8080/staff/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"staff_a1","password":"Password123!","hospital":"hospital_a"}' \
  | python3 -c 'import json,sys;print(json.load(sys.stdin)["access_token"])')

curl -s "localhost:8080/patient/search?first_name=Somchai" -H "Authorization: Bearer $TOKEN"
```

## API

Full spec: `docs/api-spec.md` (hand-written) and `docs/swagger.yaml` / Swagger UI (generated from code annotations — source of truth if the two ever disagree).

**Required by the assignment:**
- `GET /mock-his/patient/search/{id}` — standalone mock of the external Hospital A HIS system
- `POST /staff/create`, `POST /staff/login`, `GET /patient/search`

**Bonus, beyond the stated spec** (documented separately so core requirement satisfaction stays unambiguous):
- `POST /staff/refresh`, `POST /staff/logout` — short-lived (15 min) access tokens with rotating 7-day refresh tokens, DB-backed and revocable

## Deploying (Railway)

Not required by the assignment (only `docker compose` + GitHub are), but the API is deploy-ready as a single container against a managed Postgres:

1. **New Railway project** → **Add a Postgres database** (from Railway's template/plugin list — this gives you a `DATABASE_URL` you'll reference below).
2. **Add a service from this repo**, pointing at the `Dockerfile` at the repo root. No `railway.json`/build target config needed — `api` is deliberately the *last* stage in the Dockerfile, so it's what gets built by default when no `--target` is specified (see the comment in `Dockerfile`). Railway's config schema has no target/stage field at all, so this ordering is the only lever available — standard Docker/BuildKit behavior, but not something I could verify against a live Railway build from here. **After your first deploy, check the build logs confirm `api` was built (not `seed`) and that `curl <your-app>.up.railway.app/healthz` responds** before assuming this worked.
3. **Set environment variables** on the `api` service:
   - `DATABASE_URL` — reference the Postgres plugin's connection string (Railway lets you do this as `${{Postgres.DATABASE_URL}}` in their dashboard, so it stays in sync if the DB ever moves)
   - `JWT_SECRET`, `ENCRYPTION_KEY` (base64, 32 bytes — `openssl rand -base64 32`), `HMAC_SECRET` — same as local `.env`, but generate fresh values, don't reuse the dev ones from `.env.example`
   - Leave `ENABLE_DEV_CORS` unset (defaults to off)
   - Don't set `PORT` — Railway injects it automatically, and the app already reads `PORT` from the environment (`cmd/api/main.go`)
4. **Deploy.** The `api` binary self-migrates the schema on boot (`internal/dbmigrate`, idempotent — safe on every restart/redeploy), so there's no separate migration step to run.
5. **Seed data (optional)**, for a demo with sample hospitals/patients — run `cmd/seed` from your own machine against Railway's Postgres, **not** via `railway run` (that executes locally but injects the *internal* `DATABASE_URL`, e.g. `postgres.railway.internal`, which only resolves from inside Railway's own network and will fail with a DNS error from a local machine — a real, reported gotcha, not hypothetical):
   ```bash
   # One-time: on the Postgres service in the Railway dashboard, enable
   # "Public Networking" and copy the public connection string it shows you.
   DATABASE_URL="<public connection string from Railway dashboard>" \
     ENCRYPTION_KEY=<same key as the api service> \
     HMAC_SECRET=<same secret as the api service> \
     go run ./cmd/seed
   ```
   Not required for the API to function — only needed if you want the demo accounts from the [Seeded accounts](#seeded-accounts) table to exist. Turn public networking back off afterward if you don't want the DB reachable from outside Railway long-term.

`nginx` isn't part of the Railway deployment — Railway's own edge handles TLS/routing to your container's `PORT`, so it would just be a redundant hop. It stays in `docker-compose.yml` for local dev since Nginx is one of the assignment's explicitly named tech-stack requirements.

## Design docs

- `docs/er-diagram.md` — schema + rationale (bilingual name columns, blind-index encryption)
- `docs/project-structure.md` — folder layout and layering rationale
- `docs/api-spec.md` — endpoint-by-endpoint spec

## Notable design decisions

- **Local Postgres is the source of truth** for `/patient/search`, populated via seed data at startup — not a live pass-through to the mock HIS on every request.
- **Tenant isolation**: every `Staff`/`Patient` row carries `hospital_id`; every query filters on it, derived from the caller's JWT, never from client input. `/staff/create` also requires the caller's own hospital to match the target hospital.
- **`national_id`/`passport_id` are encrypted at rest** using a blind-index pattern: AES-GCM (random nonce) for the displayable value, plus a separate deterministic HMAC-SHA256 column for exact-match search — avoids storing PII in plaintext without leaking equality patterns the way naive deterministic encryption would.
- **Bilingual name columns** (`first_name_th`/`first_name_en`, etc.) mirror the Hospital A HIS response shape; `/patient/search`'s flat `first_name` input matches against both.
- **Self-migrating binaries**: `cmd/api` and `cmd/seed` both apply pending migrations on startup (`internal/dbmigrate`, embedding `migrations/*.sql` via `go:embed` — no dependency on a file being present on disk at runtime). Idempotent, so safe on every boot/redeploy. This replaced an earlier separate `migrate` init container in `docker-compose.yml`: that pattern works for local Docker Compose but has no clean equivalent on single-container PaaS platforms (Railway, Fly, Render) that don't offer a distinct pre-deploy migration step — one mechanism that works everywhere beats two that only work in different places.

## Testing

```bash
go test ./...
```

Requires Docker running — `internal/repository`'s tests spin up a real Postgres container (`testcontainers-go`), so the full suite isn't purely in-memory; everything else is.

Three layers of tests:
- **`internal/service`**: mocked repository interfaces — business logic (tenant checks, AND/exact/partial filter construction, token rotation) isolated from any DB.
- **`internal/handler`**: full HTTP stack incl. JWT middleware, using fake in-memory repositories — request/response shape, status codes, auth enforcement.
- **`internal/repository`**: real Postgres via `testcontainers-go`, migrated with the actual `migrations/000001_init.up.sql` — the real `WHERE hospital_id = ?` scoping, bilingual `ILIKE` partial matching (including case-insensitivity), blind-index hash exact-matching, AND-combined filters, and the Postgres unique-constraint → `ErrDuplicateKey` translation under a real concurrent-insert race. This is the layer that was previously only verified by manual `curl` runs — now it has its own automated coverage.

## Known limitations

- `phone_number` and `email` are stored in plaintext (not encrypted at rest) — out of scope for this assignment; `national_id`/`passport_id` were prioritized since they're the more sensitive/regulated identifiers and don't need partial-match search.
- No role/admin concept — any authenticated staff member can create a peer staff account within their own hospital.
- Refresh token rotation has no reuse-detection cascade (kept deliberately simple): presenting an already-rotated token just fails as invalid, rather than revoking the whole token family as a compromise signal.
- `go.mod` requires Go 1.25 (pulled in as a side effect of the `swaggo/swag` toolchain, not a deliberate requirement). Works out of the box via `docker compose` (Dockerfile pinned to `golang:1.25-alpine`); running `go test ./...` locally on an older toolchain needs `GOTOOLCHAIN=auto` or a local Go 1.25 install.
- `Decrypt` failures on `national_id`/`passport_id` (e.g. from an `ENCRYPTION_KEY` rotation without re-encrypting existing rows) are logged (`internal/service/patient_service.go`) but the field is still returned as empty rather than surfaced as a request-level error — a caller sees a blank field, not a failure.
