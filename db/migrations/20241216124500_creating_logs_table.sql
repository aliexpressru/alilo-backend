-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Создание таблицы logs для хранения логов глобальных запусков
create table if not exists group_logs
(
    log_id         bigserial
        primary key,
    user_name            text      default ''               not null,
    preferred_user_name  text      default ''               not null,
    logs                 text      default ''               not null,
    created_at           timestamp default now()            not null,
    updated_at           timestamp default now()            not null,
    deleted_at           timestamp
);



-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

drop table if exists group_logs;
