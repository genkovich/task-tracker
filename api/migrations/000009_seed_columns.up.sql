-- Fixed, non-editable column set (ADR-0004) — no CRUD, chosen with the user during data-model.
INSERT INTO columns (id, board_id, name, position) VALUES
    ('019a0000-0000-7000-8000-000000000201', '019a0000-0000-7000-8000-000000000101', 'To Do', 0),
    ('019a0000-0000-7000-8000-000000000202', '019a0000-0000-7000-8000-000000000101', 'In Progress', 1),
    ('019a0000-0000-7000-8000-000000000203', '019a0000-0000-7000-8000-000000000101', 'Done', 2)
ON CONFLICT (board_id, position) DO NOTHING;
