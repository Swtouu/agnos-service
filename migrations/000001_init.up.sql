CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE hospitals (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE staff (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id   UUID NOT NULL REFERENCES hospitals(id),
    username      TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hospital_id, username)
);

CREATE TABLE patients (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id UUID NOT NULL REFERENCES hospitals(id),

    first_name_th  TEXT NOT NULL DEFAULT '',
    middle_name_th TEXT NOT NULL DEFAULT '',
    last_name_th   TEXT NOT NULL DEFAULT '',
    first_name_en  TEXT NOT NULL DEFAULT '',
    middle_name_en TEXT NOT NULL DEFAULT '',
    last_name_en   TEXT NOT NULL DEFAULT '',

    date_of_birth DATE NOT NULL,
    patient_hn    TEXT NOT NULL,

    national_id_encrypted TEXT NOT NULL DEFAULT '',
    national_id_hash      TEXT NOT NULL DEFAULT '',
    passport_id_encrypted TEXT NOT NULL DEFAULT '',
    passport_id_hash      TEXT NOT NULL DEFAULT '',

    phone_number TEXT NOT NULL DEFAULT '',
    email        TEXT NOT NULL DEFAULT '',
    gender       TEXT NOT NULL CHECK (gender IN ('M', 'F')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_patients_hospital_id ON patients (hospital_id);
CREATE INDEX idx_patients_national_id_hash ON patients (national_id_hash);
CREATE INDEX idx_patients_passport_id_hash ON patients (passport_id_hash);
CREATE INDEX idx_patients_first_name_th ON patients (first_name_th);
CREATE INDEX idx_patients_first_name_en ON patients (first_name_en);
CREATE INDEX idx_patients_last_name_th ON patients (last_name_th);
CREATE INDEX idx_patients_last_name_en ON patients (last_name_en);

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id   UUID NOT NULL REFERENCES staff(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_staff_id ON refresh_tokens (staff_id);
