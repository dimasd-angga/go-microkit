CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,
    status      TEXT NOT NULL,
    total_cents BIGINT NOT NULL,
    items       JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS orders_user_idx       ON orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS orders_status_idx     ON orders (status);
