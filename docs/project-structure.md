# Project Structure

```
.
├── cmd/
│   ├── api/            # main entrypoint — HTTP server
│   │   └── main.go
│   └── seed/            # standalone seed program (Hospital/Staff/Patient fixtures)
│       └── main.go
├── internal/
│   ├── model/            # GORM structs: Hospital, Staff, Patient, RefreshToken
│   ├── crypto/            # AES-GCM encrypt/decrypt + HMAC-SHA256 blind-index hashing (PII)
│   ├── auth/            # JWT issuance/validation + refresh-token generation/hashing — shared by service (issues) and middleware (validates), avoiding a dependency cycle between them
│   ├── dbmigrate/        # applies embedded migrations at process startup — called by both cmd/api and cmd/seed
│   ├── repository/        # GORM queries only — one file per aggregate, tenant filters live here
│   ├── service/            # business logic: auth, token issuance/rotation, search filter building
│   ├── handler/            # Gin HTTP handlers — request binding, calls service, shapes response
│   ├── middleware/        # JWT auth (parses claims, injects staff_id/hospital_id into context) + permissive dev-only CORS for webui/
│   └── mockhis/            # standalone mock of the Hospital A HIS API (hardcoded in-memory data)
├── migrations/            # golang-migrate .sql files (one pair up/down per change) + migrations.go (go:embed, so cmd/api and cmd/seed can apply them without depending on files existing on disk at runtime)
├── docs/                # this file, er-diagram.md, api-spec.md, generated docs.go/swagger.json/swagger.yaml
├── webui/                # single-file HTML/JS test console (not the graded front-end deliverable)
├── docker-compose.yml    # postgres, seed (self-migrates + seeds, init), api (self-migrates + serves), nginx
├── Dockerfile            # multi-stage: shared build stage, `seed` then `api` targets — api is last so it's the default build target with no --target flag (needed for single-container platforms like Railway)
├── nginx.conf
├── .env.example
└── go.mod
```

## Layering rationale

`handler → service → repository`, each layer depending only on the one below it:

- **handler**: Gin-specific. Binds/validates the request, calls exactly one service method, translates the result/error into an HTTP response. No business logic, no GORM.
- **service**: framework-agnostic. Owns business rules — tenant-isolation checks, AND/exact/partial filter construction, password hashing, JWT issuance, refresh-token rotation. Depends on `repository` via interfaces, so it can be unit-tested against mocks without a real DB.
- **repository**: the only layer that imports GORM/talks to Postgres. Every query here that touches `Staff` or `Patient` includes a `hospital_id` filter — this is where tenant isolation is enforced at the SQL level, deliberately kept auditable even through GORM's query builder. Depends only on `model` — never on `service` — so the dependency direction stated above actually holds; query-shape types like `PatientSearchFilters` live in `model` specifically so both `service` (which builds them) and `repository` (which consumes them) can share the type without either importing the other. Tested against a real Postgres container (`testcontainers-go`, see `internal/repository/main_test.go`) rather than mocks — this is the one layer where mocking would just be testing the mock, since the whole point is to verify the actual SQL.

`mockhis` is intentionally outside this chain — it's a separate, self-contained package simulating an external system, not part of the middleware's own layering.

## Why this over alternatives

Considered a flatter structure (everything in `main.go` / a couple of files) — rejected because the assignment is explicitly evaluated on code quality/structure, and the layering directly maps to the "requirement satisfaction" and "unit test coverage" criteria (mocked repository interfaces require the service layer to depend on interfaces, not concrete GORM calls).
