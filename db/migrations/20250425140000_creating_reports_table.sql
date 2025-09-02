-- +goose Up
-- SQL in this section is executed when the migration is applied.

create type run_report_status as enum ('STATUS_UNSPECIFIED', 'STATUS_COMPLETE', 'STATUS_PROCESSING', 'STATUS_NOT_APPLIED', 'STATUS_FAILED');

-- Создание таблицы logs для хранения логов глобальных запусков
create table if not exists run_report
(
    id                      bigserial
        primary key,
    run_id                  bigint                                              not null
        CONSTRAINT run_report
        REFERENCES runs(run_id)
        ON DELETE CASCADE,

    link                    text                default ''                      not null,
    status                  run_report_status   default 'STATUS_UNSPECIFIED'    not null,

    user_name               text                default ''                      not null,
    preferred_user_name     text                default ''                      not null,

    created_at              timestamp           default now()                   not null,
    updated_at              timestamp           default now()                   not null,
    deleted_at              timestamp
);

comment on type run_report_status is 'The build status of the raw data for the Run report';
comment on table run_report is 'Table of raw data for the Run report';

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

drop table if exists run_report;
drop type if exists run_report_status cascade;
