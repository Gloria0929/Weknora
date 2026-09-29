package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *types.Project) error
	Get(ctx context.Context, tenantID uint64, id string) (*types.Project, error)
	ListByOwner(ctx context.Context, tenantID uint64, ownerID string) ([]*types.Project, error)
	Update(ctx context.Context, project *types.Project) error
	Delete(ctx context.Context, tenantID uint64, id string) error
}

type ProjectFolderRepository interface {
	Create(ctx context.Context, folder *types.ProjectFolder) error
	Get(ctx context.Context, tenantID uint64, id string) (*types.ProjectFolder, error)
	ListByProject(ctx context.Context, tenantID uint64, projectID, ownerID string) ([]*types.ProjectFolder, error)
	Delete(ctx context.Context, tenantID uint64, id string) error
}

type ProjectUpdate struct {
	Name          *string
	Description   *string
	DefaultModel  *string
	DefaultEditor *string
	GitEnabled    *bool
	Settings      *types.JSONMap
}

type ProjectFolderService interface {
	Create(ctx context.Context, tenantID uint64, ownerID, projectID string, folder *types.ProjectFolder) (*types.ProjectFolder, error)
	Get(ctx context.Context, tenantID uint64, ownerID, projectID, folderID string) (*types.ProjectFolder, error)
	List(ctx context.Context, tenantID uint64, ownerID, projectID string) ([]*types.ProjectFolder, error)
	Delete(ctx context.Context, tenantID uint64, ownerID, projectID, folderID string) error
}

// ProjectService is a domain service for Coding-mode project metadata. It
// deliberately does not run an Agent or own an Agent loop.
type ProjectService interface {
	Create(ctx context.Context, tenantID uint64, ownerID string, project *types.Project) (*types.Project, error)
	Get(ctx context.Context, tenantID uint64, ownerID, id string) (*types.Project, error)
	List(ctx context.Context, tenantID uint64, ownerID string) ([]*types.Project, error)
	Update(ctx context.Context, tenantID uint64, ownerID, id string, update ProjectUpdate) (*types.Project, error)
	Delete(ctx context.Context, tenantID uint64, ownerID, id string) error
}
