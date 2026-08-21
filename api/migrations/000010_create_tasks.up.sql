CREATE TABLE IF NOT EXISTS tasks (
    id         UUID        PRIMARY KEY,
    column_id  UUID        NOT NULL REFERENCES columns(id),
    title      VARCHAR(200) NOT NULL,
    assignee   VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tasks_column_id ON tasks (column_id);
