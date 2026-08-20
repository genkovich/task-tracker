CREATE TABLE IF NOT EXISTS boards (
    id         UUID        PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
