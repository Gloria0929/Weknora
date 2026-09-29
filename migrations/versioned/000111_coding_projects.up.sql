-- Project metadata and coding-thread session linkage. Projects reference a
-- local directory but do not grant filesystem access by themselves.
CREATE TABLE IF NOT EXISTS projects (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    owner_id VARCHAR(255) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL DEFAULT '',
    root_path VARCHAR(1024) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    default_model VARCHAR(64) NOT NULL DEFAULT '',
    default_editor VARCHAR(64) NOT NULL DEFAULT '',
    git_enabled BOOLEAN NOT NULL DEFAULT false,
    knowledge_base_ids JSONB NOT NULL DEFAULT '[]',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_projects_tenant_owner ON projects (tenant_id, owner_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects (deleted_at) WHERE deleted_at IS NOT NULL;

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS mode VARCHAR(16) NOT NULL DEFAULT 'chat';
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS project_id VARCHAR(36) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions (project_id) WHERE project_id <> '';

COMMENT ON TABLE projects IS 'User-owned Coding-mode project metadata; source files are never stored here';
COMMENT ON COLUMN projects.root_path IS 'Local directory reference, not a filesystem authorization grant';
COMMENT ON COLUMN sessions.mode IS 'chat or coding; both modes reuse the same WeKnora Agent Loop';
