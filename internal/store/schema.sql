CREATE TABLE IF NOT EXISTS tenants (
    id        BIGSERIAL PRIMARY KEY,
    name      TEXT NOT NULL,
    key_hash  TEXT NOT NULL UNIQUE,
    role      TEXT NOT NULL DEFAULT 'member',
    plan      TEXT NOT NULL DEFAULT 'standard'
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id               BIGSERIAL PRIMARY KEY,
    tenant_id        BIGINT NOT NULL REFERENCES tenants(id),
    target_url       TEXT NOT NULL,
    event_types      TEXT NOT NULL,
    secret           TEXT NOT NULL,
    header_name      TEXT NOT NULL DEFAULT '',
    header_value     TEXT NOT NULL DEFAULT '',
    docs_url         TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deliveries (
    id               BIGSERIAL PRIMARY KEY,
    subscription_id  BIGINT NOT NULL REFERENCES subscriptions(id),
    tenant_id        BIGINT NOT NULL,
    event_type       TEXT NOT NULL,
    payload          TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    attempts         INT NOT NULL DEFAULT 0,
    response_code    INT NOT NULL DEFAULT 0,
    last_error       TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deliveries_archive (LIKE deliveries INCLUDING ALL);
