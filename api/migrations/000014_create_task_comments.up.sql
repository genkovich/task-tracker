-- Коментарі задачі (tasks TSK-08). author — вільний текст без FK на users:
-- акаунтів на рівні дошки немає (ADR-0001 фічі board), форма лише предзаповнює
-- поле іменем залогіненого. ON DELETE CASCADE — це TSK-11: коментарі не
-- переживають свою задачу.
CREATE TABLE IF NOT EXISTS task_comments (
    id         UUID          PRIMARY KEY,
    task_id    UUID          NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author     VARCHAR(200)  NOT NULL,
    body       VARCHAR(2000) NOT NULL,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

-- Покриває і FK (перша колонка), і єдиний реальний запит — коментарі задачі за
-- часом. updated_at немає: коментар не редагується (spec §3).
CREATE INDEX IF NOT EXISTS idx_task_comments_task_id_created_at ON task_comments (task_id, created_at);
