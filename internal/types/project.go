package types

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Project is a user-owned Codex-style container. It is separate from
// knowledge bases and from local folders; those folders are child resources
// so one project can organize several local workspaces and their chats.
type Project struct {
	ID       string `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID uint64 `json:"tenant_id" gorm:"not null;index:idx_projects_tenant_owner"`
	OwnerID  string `json:"-" gorm:"type:varchar(255);not null;default:'';index:idx_projects_tenant_owner"`
	Name     string `json:"name" gorm:"type:varchar(255);not null;default:''"`
	// RootPath is retained only for rows created by the initial Coding preview.
	// New projects store paths exclusively in ProjectFolder.
	RootPath    string `json:"root_path,omitempty" gorm:"type:varchar(1024);not null;default:''"`
	Description string `json:"description" gorm:"type:text;not null;default:''"`

	DefaultModel  string         `json:"default_model" gorm:"type:varchar(64);not null;default:''"`
	DefaultEditor string         `json:"default_editor" gorm:"type:varchar(64);not null;default:''"`
	GitEnabled    bool           `json:"git_enabled" gorm:"not null;default:false"`
	Settings      JSONMap        `json:"settings" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Project) TableName() string { return "projects" }

func (p *Project) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	p.Name = strings.TrimSpace(p.Name)
	p.RootPath = strings.TrimSpace(p.RootPath)
	p.Description = strings.TrimSpace(p.Description)
	p.DefaultModel = strings.TrimSpace(p.DefaultModel)
	p.DefaultEditor = strings.TrimSpace(p.DefaultEditor)
	if p.Settings == nil {
		p.Settings = JSONMap{}
	}
	return nil
}

// ProjectFolder is an authorized-candidate local directory under a Coding
// project. It is metadata only: creating it never grants filesystem access.
// A Coding Thread rechecks host authorization before it can use LocalPath.
type ProjectFolder struct {
	ID        string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	ProjectID string         `json:"project_id" gorm:"type:varchar(36);not null;index:idx_project_folders_project"`
	TenantID  uint64         `json:"tenant_id" gorm:"not null;index:idx_project_folders_owner"`
	OwnerID   string         `json:"-" gorm:"type:varchar(255);not null;default:'';index:idx_project_folders_owner"`
	Name      string         `json:"name" gorm:"type:varchar(255);not null;default:''"`
	LocalPath string         `json:"local_path" gorm:"type:varchar(1024);not null;default:''"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ProjectFolder) TableName() string { return "project_folders" }

func (f *ProjectFolder) BeforeCreate(_ *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	f.Name = strings.TrimSpace(f.Name)
	f.LocalPath = strings.TrimSpace(f.LocalPath)
	return nil
}
