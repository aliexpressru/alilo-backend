-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- добавление представления для общего поиска
DROP VIEW IF EXISTS view_scripts CASCADE;

create or replace view view_search as
select scripts.script_id as id,
       'script' as type,
       scripts.project_id as project_id,
       scripts.scenario_id as scenario_id,
       concat(scripts.script_id, '|-|',
              scripts.name, '|-|',
              scripts.descrip, '|-|',
              scripts.project_id, '|-|',
              scripts.scenario_id, '|-|',
              scripts.enabled, '|-|',
              scripts.tag, '|-|',
              scripts.script_file, '|-|',
              scripts.ammo_id, '|-|',
              scripts.base_url, '|-|',
              scripts.options_rps, '|-|',
              scripts.options_steps, '|-|',
              scripts.options_duration, '|-|',
              scripts.grafana_url, '|-|',
              scripts.additional_env, '|-|',
              scripts.cmt_rps, '|-|',
              scripts.expr_rps, '|-|',
              scripts.cmt_rt, '|-|',
              scripts.expr_rt, '|-|',
              scripts.cmt_err, '|-|',
              scripts.expr_err, '|-|') as data
from scripts
UNION ALL
SELECT simple_scripts.script_id as id,
                 'simple_script' as type,
                 simple_scripts.project_id,
                 simple_scripts.scenario_id,
                 concat(simple_scripts.script_id, '|-|',
                        simple_scripts.name, '|-|',
                        simple_scripts.description, '|-|',
                        simple_scripts.project_id, '|-|',
                        simple_scripts.scenario_id, '|-|',
                        simple_scripts.enabled, '|-|',
                        simple_scripts.tag, '|-|',
                        simple_scripts.scheme, '|-|',
                        simple_scripts.path, '|-|',
                        simple_scripts.http_method, '|-|',
                        simple_scripts.script_file_url, '|-|',
                        simple_scripts.static_ammo, '|-|',
                        simple_scripts.ammo_url, '|-|',
                        simple_scripts.is_static_ammo, '|-|',
                        simple_scripts.rps, '|-|',
                        simple_scripts.duration, '|-|',
                        simple_scripts.steps, '|-|',
                        simple_scripts.query_params, '|-|',
                        simple_scripts.headers, '|-|',
                        simple_scripts.additional_env, '|-|',
                        simple_scripts.max_v_us, '|-|',
                        simple_scripts.monitoring_links, '|-|',
                        simple_scripts.cmt_rps, '|-|',
                        simple_scripts.expr_rps, '|-|',
                        simple_scripts.cmt_rt, '|-|',
                        simple_scripts.expr_rt, '|-|',
                        simple_scripts.cmt_err, '|-|',
                        simple_scripts.expr_err, '|-|') as data
from simple_scripts
UNION ALL
SELECT scenarios.scenario_id as id,
                 'scenario' as type,
                 scenarios.project_id  as project_id,
                 scenarios.scenario_id  as scenario_id,
                 concat(scenarios.scenario_id, '|-|',
                        scenarios.title, '|-|',
                        scenarios.descrip, '|-|',
                        scenarios.project_id, '|-|',
                        scenarios.selectors, '|-|') as data
from scenarios
UNION ALL
SELECT projects.id as id,
                 'project' as type,
       projects.id as project_id,
       '-1' as scenario_id,
                 concat(projects.id, '|-|',
                        projects.title, '|-|',
                        projects.descrip, '|-|') as data
from projects;

comment on view view_search is 'A "view" for searching through all load entities';

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP VIEW IF EXISTS view_search CASCADE;
