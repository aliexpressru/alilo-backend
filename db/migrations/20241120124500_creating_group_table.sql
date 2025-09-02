-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Добавление таблиц statistic и statistic_dump для хранения статистических данных по обстрелам
create table if not exists groups_page
(
    page_id         bigserial
        primary key,
    user_name            text      default ''               not null,
    preferred_user_name  text      default ''               not null,
    groups               text      default '{}'::text[]     not null,
    created_at           timestamp default now()            not null,
    updated_at           timestamp default now()            not null,
    deleted_at           timestamp
);


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

drop table if exists groups_page;
