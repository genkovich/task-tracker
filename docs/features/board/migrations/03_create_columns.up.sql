CREATE TABLE IF NOT EXISTS columns (
    id         UUID        PRIMARY KEY,
    board_id   UUID        NOT NULL REFERENCES boards(id),
    name       VARCHAR(100) NOT NULL,
    position   SMALLINT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (board_id, position)
);

CREATE INDEX IF NOT EXISTS idx_columns_board_id_position ON columns (board_id, position);
