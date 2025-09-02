-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Alter columns to change ENUM to TEXT
alter table if exists scripts
    alter column source_rps type text using source_rps::text;

alter table if exists scripts
    alter column source_rps set default '';


alter table scripts
    alter column source_rt type text using source_rt::text;

alter table scripts
    alter column source_rt set default '';

alter table scripts
    alter column source_err type text using source_err::text;

alter table scripts
    alter column source_err set default '';

alter table scripts
    alter column source_rps type text using source_rps::text;

alter table simple_scripts
    alter column source_rps type text using source_rps::text;

alter table simple_scripts
    alter column source_rps set default '';

alter table simple_scripts
    alter column source_rt type text using source_rt::text;

alter table simple_scripts
    alter column source_rt set default '';

alter table simple_scripts
    alter column source_err type text using source_err::text;

alter table simple_scripts
    alter column source_err set default '';

-- Drop ENUM type
DROP TYPE IF EXISTS e_source_vmc CASCADE;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

-- Create ENUM type
CREATE TYPE e_source_vmc AS ENUM (
    'DATASOURCE_VMC_UNSPECIFIED',
    'DATASOURCE_VMC_HC'
);

-- Alter columns to change TEXT back to ENUM
ALTER TABLE IF EXISTS simple_scripts
ALTER COLUMN source_err TYPE e_source_vmc USING source_err::e_source_vmc;

ALTER TABLE IF EXISTS simple_scripts
ALTER COLUMN source_rt TYPE e_source_vmc USING source_rt::e_source_vmc;

ALTER TABLE IF EXISTS simple_scripts
ALTER COLUMN source_rps TYPE e_source_vmc USING source_rps::e_source_vmc;

ALTER TABLE IF EXISTS scripts
ALTER COLUMN source_err TYPE e_source_vmc USING source_err::e_source_vmc;

ALTER TABLE IF EXISTS scripts
ALTER COLUMN source_rt TYPE e_source_vmc USING source_rt::e_source_vmc;

ALTER TABLE IF EXISTS scripts
ALTER COLUMN source_rps TYPE e_source_vmc USING source_rps::e_source_vmc;