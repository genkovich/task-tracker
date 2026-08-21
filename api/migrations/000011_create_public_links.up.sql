CREATE TABLE IF NOT EXISTS public_links (
    id         UUID        PRIMARY KEY,
    board_id   UUID        NOT NULL UNIQUE REFERENCES boards(id),
    token      VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
