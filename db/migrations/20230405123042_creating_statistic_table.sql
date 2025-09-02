-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление таблиц statistic и statistic_dump для хранения статистических данных по обстрелам
create table if not exists statistic_dump
(
    statistic_dump_id         bigserial
        primary key,

    created_at timestamp default now() not null
);

create table if not exists statistic
(
    statistic_id        bigserial
        primary key,
    statistic_dump_id   bigint                  not null
        constraint statistic_dump_fkey
            references statistic_dump,

    url_path    text                        not null,
    url_method   text                        not null,

    project_ids         bigint[]    default ARRAY []::bigint[]      not null,
    scenario_ids        bigint[]    default ARRAY []::bigint[]      not null,
    run_ids             bigint[]    default ARRAY []::bigint[]      not null,
    script_run_ids      bigint[]    default ARRAY []::bigint[]      not null,
    script_ids          bigint[]    default ARRAY []::bigint[]      not null,
    trace_ids           text[]      default ARRAY []::text[]      not null,
    agents              text[]      default ARRAY []::text[]      not null,

    rps                         bigint                      not null,
    rt_90_p                     bigint                      not null,
    rt_95_p                     bigint                      not null,
    rt_99_p                     bigint                      not null,
    rt_max                      bigint                      not null,
    vus                         bigint                      not null,
    failed                      bigint                      not null,
    data_sent                   bigint                      not null,
    data_received               bigint                      not null,
    current_test_run_duration   text                        not null

);

