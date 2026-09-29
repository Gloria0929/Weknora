-- Mirrors versioned migration 000111_coding_projects.
CREATE TABLE IF NOT EXISTS projects (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    owner_id VARCHAR(255) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL DEFAULT '',
    root_path VARCHAR(1024) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    default_model VARCHAR(64) NOT NULL DEFAULT '',
    default_editor VARCHAR(64) NOT NULL DEFAULT '',
    git_enabled BOOLEAN NOT NULL DEFAULT 0,
    knowledge_base_ids TEXT NOT NULL DEFAULT '[]',
    settings TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_projects_tenant_owner ON projects (tenant_id, owner_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects (deleted_at);

ALTER TABLE sessions ADD COLUMN mode VARCHAR(16) NOT NULL DEFAULT 'chat';
ALTER TABLE sessions ADD COLUMN project_id VARCHAR(36) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions (project_id);
