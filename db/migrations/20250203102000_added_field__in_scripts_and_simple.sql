-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление поля "additional_env" в таблицы scripts и simple_scripts
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS additional_env TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS additional_env TEXT NOT NULL DEFAULT '';


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS additional_env;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS additional_env;
