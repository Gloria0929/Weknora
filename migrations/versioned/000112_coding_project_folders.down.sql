DROP INDEX IF EXISTS idx_sessions_project_folder_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS project_folder_id;
DROP TABLE IF EXISTS project_folders;
