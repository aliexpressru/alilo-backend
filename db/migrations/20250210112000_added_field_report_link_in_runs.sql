-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление поля "report_link" в таблицы runs
ALTER TABLE IF EXISTS runs
    ADD COLUMN IF NOT EXISTS report_link TEXT;


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE IF EXISTS runs
    DROP COLUMN IF EXISTS report_link;
