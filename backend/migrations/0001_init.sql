-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Profissionais (sem PII do paciente)
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    plan          TEXT        NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'pro')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);

-- Assinaturas (vínculo Stripe / ciclo)
CREATE TABLE subscriptions (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type                   TEXT        NOT NULL CHECK (type IN ('free', 'pro')),
    status                 TEXT        NOT NULL CHECK (status IN ('active', 'canceled', 'past_due')),
    stripe_customer_id     TEXT,
    stripe_subscription_id TEXT,
    current_period_end     TIMESTAMPTZ
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions (user_id);

-- Cotas de uso mensais (Free: ≤10/mês; Pro: sem limite)
CREATE TABLE usage_quotas (
    id      UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    period  TEXT    NOT NULL,        -- formato 'YYYY-MM'
    count   INTEGER NOT NULL DEFAULT 0 CHECK (count >= 0),
    UNIQUE (user_id, period)
);

-- Evoluções clínicas (Pro persiste; Free é efêmero — sem PII do paciente)
CREATE TABLE evolutions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    label            TEXT,                      -- rótulo interno do profissional, sem PII
    soap_s           TEXT        NOT NULL DEFAULT '',
    soap_o           TEXT        NOT NULL DEFAULT '',
    soap_a           TEXT        NOT NULL DEFAULT '',
    soap_p           TEXT        NOT NULL DEFAULT '',
    cid_suggestions  JSONB       NOT NULL DEFAULT '[]',
    confidence_flags JSONB       NOT NULL DEFAULT '[]',
    status           TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'finalized')),
    source_refs      JSONB       NOT NULL DEFAULT '[]',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_evolutions_user_id_created ON evolutions (user_id, created_at DESC);

-- Fontes clínicas para RAG (CREFITO, protocolos)
CREATE TABLE clinical_sources (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT        NOT NULL,
    origin      TEXT        NOT NULL,  -- ex.: 'CREFITO', 'protocolo-X'
    version     TEXT        NOT NULL,
    chunk_text  TEXT        NOT NULL,
    embedding   VECTOR(1536),          -- dimensão ajustável via EMBEDDING_DIM
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_clinical_sources_embedding ON clinical_sources
    USING hnsw (embedding vector_cosine_ops);

-- Audit log de acesso (LGPD — sem PII do paciente)
CREATE TABLE audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        REFERENCES users (id) ON DELETE SET NULL,
    action      TEXT        NOT NULL,
    resource_id UUID,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS clinical_sources;
DROP TABLE IF EXISTS evolutions;
DROP TABLE IF EXISTS usage_quotas;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "vector";
DROP EXTENSION IF EXISTS "pgcrypto";
-- +goose StatementEnd
