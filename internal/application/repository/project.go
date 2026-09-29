package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type projectRepository struct{ db *gorm.DB }

func NewProjectRepository(db *gorm.DB) interfaces.ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(ctx context.Context, project *types.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *projectRepository) Get(ctx context.Context, tenantID uint64, id string) (*types.Project, error) {
	var project types.Project
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) ListByOwner(ctx context.Context, tenantID uint64, ownerID string) ([]*types.Project, error) {
	var projects []*types.Project
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND owner_id = ?", tenantID, ownerID).
		Order("updated_at DESC").Find(&projects).Error
	return projects, err
}

func (r *projectRepository) Update(ctx context.Context, project *types.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *projectRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&types.Project{}).Error
}

type projectFolderRepository struct{ db *gorm.DB }

func NewProjectFolderRepository(db *gorm.DB) interfaces.ProjectFolderRepository {
	return &projectFolderRepository{db: db}
}

func (r *projectFolderRepository) Create(ctx context.Context, folder *types.ProjectFolder) error {
	return r.db.WithContext(ctx).Create(folder).Error
}

func (r *projectFolderRepository) Get(ctx context.Context, tenantID uint64, id string) (*types.ProjectFolder, error) {
	var folder types.ProjectFolder
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&folder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (r *projectFolderRepository) ListByProject(ctx context.Context, tenantID uint64, projectID, ownerID string) ([]*types.ProjectFolder, error) {
	var folders []*types.ProjectFolder
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND project_id = ? AND owner_id = ?", tenantID, projectID, ownerID).
		Order("updated_at DESC").Find(&folders).Error
	return folders, err
}

func (r *projectFolderRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&types.ProjectFolder{}).Error
}
