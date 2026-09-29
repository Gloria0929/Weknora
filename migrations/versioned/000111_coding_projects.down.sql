DROP INDEX IF EXISTS idx_sessions_project_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS project_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS mode;
DROP INDEX IF EXISTS idx_projects_deleted_at;
DROP INDEX IF EXISTS idx_projects_tenant_owner;
DROP TABLE IF EXISTS projects;
