-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление таблиц traces для хранения трейсов
create table IF NOT EXISTS traces
(
    id            bigserial
        primary key,
    url_paths     text[]    default ARRAY []::text[]   not null,
    trace_id      text                                 not null,
    project_id    bigint                               not null,
    scenario_id   bigint                               not null,
    script_id     bigint                               not null,
    run_id        bigint                               not null,
    run_script_id bigint                               not null,
    agent         text                                 not null,
    trace_time    timestamp default now()              not null
);

