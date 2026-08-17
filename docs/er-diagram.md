# ER Diagram

```mermaid
erDiagram
    HOSPITAL ||--o{ STAFF : employs
    HOSPITAL ||--o{ PATIENT : "has records for"
    STAFF ||--o{ REFRESH_TOKEN : holds

    HOSPITAL {
        uuid id PK
        string code UK "lookup key, e.g. hospital_a"
        string name
        timestamp created_at
    }

    STAFF {
        uuid id PK
        uuid hospital_id FK
        string username "unique per (hospital_id, username)"
        string password_hash "bcrypt"
        timestamp created_at
    }

    PATIENT {
        uuid id PK
        uuid hospital_id FK
        string first_name_th
        string middle_name_th
        string last_name_th
        string first_name_en
        string middle_name_en
        string last_name_en
        date date_of_birth
        string patient_hn
        bytea national_id_encrypted "AES-GCM, decrypt for display"
        string national_id_hash UK "HMAC-SHA256, exact-match lookup"
        bytea passport_id_encrypted "AES-GCM, decrypt for display"
        string passport_id_hash UK "HMAC-SHA256, exact-match lookup"
        string phone_number
        string email
        string gender "M or F"
        timestamp created_at
    }

    REFRESH_TOKEN {
        uuid id PK
        uuid staff_id FK
        string token_hash "SHA-256 of the raw refresh token"
        timestamp expires_at
        timestamp revoked_at "nullable"
        timestamp created_at
    }
```

## Notes

- **Bilingual name columns**: `Patient` mirrors the Hospital A HIS response shape (`first_name_th`/`first_name_en`, etc.) rather than a single flattened name, per task 2's "compatible with hospital data structures" requirement. `/patient/search`'s flat `first_name`/`middle_name`/`last_name` input matches against both the `_th` and `_en` columns.
- **Blind-index encryption**: `national_id` and `passport_id` are stored encrypted (AES-GCM, random nonce) for display, with a separate deterministic HMAC-SHA256 hash column used for exact-match search (`WHERE national_id_hash = ?`). This avoids storing PII in plaintext while keeping equality lookup possible, without leaking equality patterns the way naive deterministic encryption would. See `internal/crypto`.
- **Tenant isolation**: every `Staff` and `Patient` row carries `hospital_id`. All patient search queries and staff-creation authorization checks filter/compare against the caller's `hospital_id`, derived from JWT claims — never from client-supplied input.
- **RefreshToken**: stores only a hash of the refresh token (never the raw value), scoped to a `staff_id`. `/staff/refresh` rotates (deletes old, inserts new); `/staff/logout` deletes the current one. No reuse-detection cascade (deliberately kept simple).
