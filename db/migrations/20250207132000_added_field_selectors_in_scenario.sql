-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление поля "selectors" в таблицы scenarios
ALTER TABLE IF EXISTS scenarios
    ADD COLUMN IF NOT EXISTS selectors TEXT NOT NULL DEFAULT '[]';


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE IF EXISTS scenarios
    DROP COLUMN IF EXISTS selectors;
