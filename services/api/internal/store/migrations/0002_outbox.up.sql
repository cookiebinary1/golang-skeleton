CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    subject TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
