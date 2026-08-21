CREATE TABLE IF NOT EXISTS cards (
    id            UUID         PRIMARY KEY,
    name          VARCHAR(200) NOT NULL,
    assignee      VARCHAR(100),
    column_status VARCHAR(20)  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
