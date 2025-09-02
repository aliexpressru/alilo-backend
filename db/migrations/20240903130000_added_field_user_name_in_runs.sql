-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление поля "user_name" в таблицу runs, требуется для получения
ALTER TABLE IF EXISTS runs
    ADD COLUMN IF NOT EXISTS user_name TEXT NOT NULL DEFAULT '-';


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE IF EXISTS runs
    DROP COLUMN IF EXISTS user_name;
