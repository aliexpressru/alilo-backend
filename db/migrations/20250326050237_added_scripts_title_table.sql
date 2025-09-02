-- +goose Up
-- +goose StatementBegin
-- Add the new column to scripts table
ALTER TABLE scripts
    ADD COLUMN title VARCHAR(255);

-- Initialize the title field with values from the name field in scripts table
UPDATE scripts
SET title = name
WHERE title IS NULL;

-- Make the title column NOT NULL in scripts table
ALTER TABLE scripts
    ALTER COLUMN title SET NOT NULL;

-- Add the new column to simple_scripts table
ALTER TABLE simple_scripts
    ADD COLUMN title VARCHAR(255);

-- Initialize the title field with values from the name field in simple_scripts table
UPDATE simple_scripts
SET title = name
WHERE title IS NULL;

-- Make the title column NOT NULL in simple_scripts table
ALTER TABLE simple_scripts
    ALTER COLUMN title SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove the title column from scripts table
ALTER TABLE scripts
DROP COLUMN title;

-- Remove the title column from simple_scripts table
ALTER TABLE simple_scripts
DROP COLUMN title;
-- +goose StatementEnd
