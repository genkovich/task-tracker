-- Назва дошки з дашборда (BRD-02). DEFAULT тут — backfill для єдиного
-- існуючого seed-рядка («перша дошка»), не постійний бізнес-default:
-- одразу знімається, name завжди задає застосунок.
ALTER TABLE boards ADD COLUMN IF NOT EXISTS name VARCHAR(200) NOT NULL DEFAULT 'Дошка команди';
ALTER TABLE boards ALTER COLUMN name DROP DEFAULT;
