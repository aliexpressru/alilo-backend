-- +goose Up

ALTER TABLE scripts
    ALTER COLUMN additional_env SET DEFAULT '{}';
UPDATE scripts
SET additional_env = '{}'
WHERE additional_env = '';

ALTER TABLE simple_scripts
    ALTER COLUMN additional_env SET DEFAULT '{}';
UPDATE simple_scripts
SET additional_env = '{}'
WHERE additional_env = '';

-- +goose Down
ALTER TABLE scripts
    ALTER COLUMN additional_env SET DEFAULT '';
UPDATE scripts
SET additional_env = ''
WHERE additional_env = '{}';

ALTER TABLE simple_scripts
    ALTER COLUMN additional_env SET DEFAULT '';
UPDATE simple_scripts
SET additional_env = ''
WHERE additional_env = '{}';

