-- Codex-style hierarchy: Project -> local Folder -> Coding Thread.
CREATE TABLE IF NOT EXISTS project_folders (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    tenant_id INTEGER NOT NULL,
    owner_id VARCHAR(255) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL DEFAULT '',
    local_path VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_project_folders_project ON project_folders (project_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_folders_owner ON project_folders (tenant_id, owner_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_folders_deleted_at ON project_folders (deleted_at);

ALTER TABLE sessions ADD COLUMN project_folder_id VARCHAR(36) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_project_folder_id ON sessions (project_folder_id);
