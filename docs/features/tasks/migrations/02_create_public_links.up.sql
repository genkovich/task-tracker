CREATE TABLE IF NOT EXISTS public_links (
    id          UUID         PRIMARY KEY,
    token       VARCHAR(255) NOT NULL UNIQUE,
    disabled_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
