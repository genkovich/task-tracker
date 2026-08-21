-- Деталі задачі (tasks TSK-01/TSK-03/TSK-05). DEFAULT тут лишається постійно:
-- '' і 'medium' структурно чесні для задачі, створеної без деталей («опису ще
-- немає», «звичайний пріоритет»), тож single-step ALTER достатній.
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS priority VARCHAR(10) NOT NULL DEFAULT 'medium';
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS due_date DATE;

-- Друга лінія оборони поверх доменної валідації (як CHECK на status-подібних
-- колонках у .claude/rules/migrations.md §Allowed) — первинний валідатор далі
-- домен, тут лише страховка від запису повз нього.
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_priority_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_priority_check CHECK (priority IN ('low', 'medium', 'high'));
