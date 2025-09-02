-- +goose Up
-- SQL in this section is executed when the migration is applied.

create type estatus as enum ('STATUS_STOPPED_UNSPECIFIED', 'STATUS_PREPARED', 'STATUS_RUNNING', 'STATUS_SCHEDULED', 'STATUS_STOPPING');

create type cmdtype as enum ('TYPE_RUN_SCENARIO_UNSPECIFIED', 'TYPE_STOP_SCENARIO', 'TYPE_RUN_SCRIPT', 'TYPE_STOP_SCRIPT', 'TYPE_UPDATE', 'TYPE_ADJUSTMENT', 'TYPE_INCREASE', 'TYPE_RUN_SIMPLE_SCRIPT');

create type cmdscope as enum ('SCOPE_ALL_UNSPECIFIED', 'SCOPE_ID');

create type cmdstatus as enum ('STATUS_CREATED_UNSPECIFIED', 'STATUS_PROCESSED', 'STATUS_FAILED', 'STATUS_COMPLETED');

create type escheme as enum ('http', 'https', '');

create type ehttp_method as enum ('get', 'post', 'head', 'put', 'patch', 'del');

create table if not exists projects
(
    id         bigserial
        primary key,
    title      varchar(300)            not null,
    descrip    text,
    created_at timestamp default now() not null,
    updated_at timestamp default now() not null,
    deleted_at timestamp
);

create table if not exists scenarios
(
    scenario_id bigserial
        primary key,
    project_id  bigint                  not null
        constraint scenarios_projects_fkey
            references projects
            on delete cascade,
    title       varchar(300)            not null,
    descrip     text,
    created_at  timestamp default now() not null,
    updated_at  timestamp default now() not null,
    deleted_at  timestamp
);

create table if not exists scripts
(
    script_id        bigserial
        primary key,
    name             varchar(300)                   not null,
    descrip          text,
    project_id       bigint                         not null,
    scenario_id      bigint                         not null
        constraint scripts_scenarios_fkey
            references scenarios
            on delete cascade,
    script_file      varchar(300),
    ammo_id          varchar(300),
    base_url         varchar(300),
    options_rps      integer,
    options_steps    integer,
    options_duration text,
    created_at       timestamp default now()        not null,
    updated_at       timestamp,
    deleted_at       timestamp,
    enabled          boolean   default false        not null,
    grafana_url      text[]    default '{}'::text[] not null,
    tag              text      default ''::text     not null
);

create table if not exists runs
(
    run_id               bigserial
        primary key,
    title                varchar(300)               not null,
    project_id           bigint                     not null,
    scenario_id          bigint                     not null,
    status               estatus                    not null,
    script_runs          text                       not null,
    created_at           timestamp default now()    not null,
    updated_at           timestamp default now()    not null,
    deleted_at           timestamp,
    info                 text      default ''::text not null,
    percentage_of_target bigint    default 0        not null
);

create table if not exists agent
(
    agent_id      bigserial
        primary key,
    host_name     text      default 'TYPE_RUN_SCENARIO_UNSPECIFIED'::text not null,
    port          text      default 'SCOPE_ALL_UNSPECIFIED'::text         not null,
    enabled       boolean   default false                                 not null,
    created_at    timestamp default now()                                 not null,
    updated_at    timestamp,
    deleted_at    timestamp,
    tags          text[]    default ARRAY []::text[]                      not null,
    cpu_used      bigint    default 0                                     not null,
    mem_used      bigint    default 0                                     not null,
    ports_used    bigint    default 0                                     not null,
    total_loading smallint  default 0                                     not null
);

create table if not exists command
(
    command_id           bigserial
        primary key,
    type                 cmdtype   default 'TYPE_RUN_SCENARIO_UNSPECIFIED'::cmdtype not null,
    scope                cmdscope  default 'SCOPE_ALL_UNSPECIFIED'::cmdscope        not null,
    run_id               bigint                                                     not null
        references runs
            on delete cascade,
    status               cmdstatus default 'STATUS_CREATED_UNSPECIFIED'::cmdstatus  not null,
    error_description    text                                                       not null,
    hostname             text                                                       not null,
    created_at           timestamp default now()                                    not null,
    updated_at           timestamp,
    deleted_at           timestamp,
    script_ids           bigint[]  default ARRAY []::bigint[]                       not null,
    percentage_of_target bigint    default 0,
    increase_rps         bigint    default '-1'::integer                            not null
);

create table if not exists simple_scripts
(
    script_id        bigserial
        primary key,
    name             text         default 'New Script'::text  not null,
    description      text         default ''::text            not null,
    project_id       bigint                                   not null,
    scenario_id      bigint                                   not null
        constraint scripts_scenarios_fkey
            references scenarios
            on delete cascade,
    enabled          boolean      default false               not null,
    monitoring_links text[]       default '{}'::text[]        not null,
    tag              text         default ''::text            not null,
    scheme           escheme      default ''::escheme         not null,
    path             text         default ''::text            not null,
    http_method      ehttp_method default 'get'::ehttp_method not null,
    script_file_url  text         default ''::text            not null,
    static_ammo      text         default ''::text            not null,
    ammo_url         text         default ''::text            not null,
    is_static_ammo   boolean      default false               not null,
    rps              text         default ''::text            not null,
    duration         text         default ''::text            not null,
    steps            text         default ''::text            not null,
    max_v_us         text         default ''::text            not null,
    query_params     text         default '[{}]'::text        not null,
    headers          text         default '{}'::text          not null,
    created_at       timestamp    default now()               not null,
    updated_at       timestamp,
    deleted_at       timestamp
);

create or replace view view_scripts(script_id, type, project_id, scenario_id, data) as
SELECT scripts.script_id,
       'script'::text                                                                      AS type,
       scripts.project_id,
       scripts.scenario_id,
       concat(scripts.script_id, scripts.name, scripts.descrip, scripts.project_id, scripts.scenario_id,
              scripts.script_file, scripts.ammo_id, scripts.base_url, scripts.options_rps, scripts.options_steps,
              scripts.options_duration, scripts.enabled, scripts.grafana_url, scripts.tag) AS data
FROM scripts
UNION ALL
SELECT simple_scripts.script_id,
       'simple_scripts'::text         AS type,
       simple_scripts.project_id,
       simple_scripts.scenario_id,
       concat(simple_scripts.script_id, simple_scripts.name, simple_scripts.description, simple_scripts.project_id,
              simple_scripts.scenario_id, simple_scripts.enabled, simple_scripts.monitoring_links, simple_scripts.tag,
              simple_scripts.scheme, simple_scripts.path, simple_scripts.http_method, simple_scripts.script_file_url,
              simple_scripts.static_ammo, simple_scripts.ammo_url, simple_scripts.is_static_ammo, simple_scripts.rps,
              simple_scripts.duration, simple_scripts.steps, simple_scripts.max_v_us, simple_scripts.query_params,
              simple_scripts.headers) AS data
FROM simple_scripts;

comment on view view_scripts is 'view for searching by all scripts';
