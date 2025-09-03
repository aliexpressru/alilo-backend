-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Создаём проект
INSERT INTO projects (title, descrip)
VALUES
    ('Project Alpha', 'This is the first test project.');

-- Создаём сценарий и сразу привязываем к проекту
INSERT INTO scenarios (project_id, title, descrip, selectors)
SELECT id, 'Scenario', 'First scenario for Project Alpha', '[]'
FROM projects WHERE title = 'Project Alpha';

-- Получаем ID созданного сценария
INSERT INTO simple_scripts(name,description,project_id,scenario_id,enabled,monitoring_links,tag,scheme,path,http_method,script_file_url,static_ammo,ammo_url,is_static_ammo,rps,duration,steps,max_v_us,query_params,headers,created_at,updated_at,deleted_at,expr_rps,source_rps,cmt_rps,expr_rt,source_rt,cmt_rt,expr_err,source_err,cmt_err,additional_env,title)
SELECT 'Alpha_alilo_script','example',p.id,s.scenario_id,TRUE,'{http://monitor1.ru,http://monitor2.ru}','loadtest','','google.com','get','http://minio:9000/test-data/Project%20Alpha/Scenario%20A1/Alpha_alilo_script.js','','',TRUE,'10','5m','1','300','[{"key":"x","value":"1"},{"key":"y","value":"1"}]','{"loadtest":"true","source-type":"powered-by-alilo"}','2025-08-15 09:14:52.159008','2025-08-15 09:15:24.329456',NULL,'','','','','','','','','','{}','Alpha alilo script'
FROM projects p, scenarios s 
WHERE p.title = 'Project Alpha' AND s.title = 'Scenario' AND s.project_id = p.id;

-- Добавляем агента
INSERT INTO agent (
    host_name, port, enabled, tags,
    cpu_used, mem_used, ports_used, total_loading,
    created_at, updated_at
)
VALUES
    (
        'agent',
        '8888',
        true,
        ARRAY['loadtest'],
        0,
        0,
        0,
        0,
        now(),
        now()
    );

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DELETE FROM simple_scripts
WHERE name = 'Alpha_alilo_script';

DELETE FROM scenarios
WHERE title = 'Scenario A1'
  AND project_id = (SELECT id FROM projects WHERE title = 'Project Alpha');

DELETE FROM projects
WHERE title = 'Project Alpha';

DELETE FROM agent
WHERE host_name = 'agent' AND port = '8282';
