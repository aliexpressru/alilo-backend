-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Типы data-source
CREATE TYPE e_source_vmc AS ENUM (
    'DATASOURCE_VMC_UNSPECIFIED',
    'DATASOURCE_VMC_HC'
    );


-- Добавление поля "selectors_expressions" в таблицу scripts
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS expr_rps TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS source_rps e_source_vmc NOT NULL DEFAULT 'DATASOURCE_VMC_UNSPECIFIED';
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS cmt_rps TEXT NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS expr_rt TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS source_rt e_source_vmc NOT NULL DEFAULT 'DATASOURCE_VMC_UNSPECIFIED';
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS cmt_rt TEXT NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS expr_err TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS source_err e_source_vmc NOT NULL DEFAULT 'DATASOURCE_VMC_UNSPECIFIED';
ALTER TABLE IF EXISTS scripts
    ADD COLUMN IF NOT EXISTS cmt_err TEXT NOT NULL DEFAULT '';

-- Добавление поля "selectors_expressions" в таблицу simple_scripts
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS expr_rps TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS source_rps e_source_vmc NOT NULL DEFAULT 'DATASOURCE_VMC_UNSPECIFIED';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS cmt_rps TEXT NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS expr_rt TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS source_rt e_source_vmc NOT NULL DEFAULT 'DATASOURCE_VMC_UNSPECIFIED';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS cmt_rt TEXT NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS expr_err TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS source_err e_source_vmc NOT NULL DEFAULT 'DATASOURCE_VMC_UNSPECIFIED';
ALTER TABLE IF EXISTS simple_scripts
    ADD COLUMN IF NOT EXISTS cmt_err TEXT NOT NULL DEFAULT '';



-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

-- Удаление поля "selectors_expressions" из таблицы scripts
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS expr_rps;
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS source_rps;
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS cmt_rps;

ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS expr_rt;
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS source_rt;
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS cmt_rt;

ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS expr_err;
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS source_err;
ALTER TABLE IF EXISTS scripts
    DROP COLUMN IF EXISTS cmt_err;

-- Удаление поля "selectors_expressions" из таблицу simple_scripts
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS expr_rps;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS source_rps;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS cmt_rps;

ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS expr_rt;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS source_rt;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS cmt_rt;

ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS expr_err;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS source_err;
ALTER TABLE IF EXISTS simple_scripts
    DROP COLUMN IF EXISTS cmt_err;

DROP TYPE IF EXISTS e_source_vmc CASCADE;
