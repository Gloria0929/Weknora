-- Codex-style hierarchy: Project -> local Folder -> Coding Thread.
CREATE TABLE IF NOT EXISTS project_folders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    owner_id VARCHAR(255) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL DEFAULT '',
    local_path VARCHAR(1024) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_project_folders_project ON project_folders (project_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_folders_owner ON project_folders (tenant_id, owner_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_folders_deleted_at ON project_folders (deleted_at) WHERE deleted_at IS NOT NULL;

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS project_folder_id VARCHAR(36) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_project_folder_id ON sessions (project_folder_id) WHERE project_folder_id <> '';

COMMENT ON TABLE project_folders IS 'Local directory metadata below a Coding project; this row does not grant filesystem access';
COMMENT ON COLUMN sessions.project_folder_id IS 'Folder selected for a Coding Thread';
