package service

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

var ErrProjectNotFound = errors.New("project not found")

const (
	projectMaxNameLen = 255
	projectMaxPathLen = 1024
	projectMaxDefault = 64
)

type projectService struct {
	repo interfaces.ProjectRepository
}

func NewProjectService(repo interfaces.ProjectRepository) interfaces.ProjectService {
	return &projectService{repo: repo}
}

func (s *projectService) Create(ctx context.Context, tenantID uint64, ownerID string, in *types.Project) (*types.Project, error) {
	if in == nil {
		return nil, apperrors.NewBadRequestError("project is required")
	}
	if strings.TrimSpace(ownerID) == "" {
		return nil, apperrors.NewForbiddenError("a signed-in user is required")
	}
	name, err := normalizeProjectName(in.Name)
	if err != nil {
		return nil, err
	}
	model, editor, err := normalizeProjectDefaults(in.DefaultModel, in.DefaultEditor)
	if err != nil {
		return nil, err
	}
	row := &types.Project{TenantID: tenantID, OwnerID: ownerID, Name: name,
		Description: strings.TrimSpace(in.Description), DefaultModel: model, DefaultEditor: editor,
		GitEnabled: in.GitEnabled, Settings: in.Settings}
	if row.Settings == nil {
		row.Settings = types.JSONMap{}
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *projectService) Get(ctx context.Context, tenantID uint64, ownerID, id string) (*types.Project, error) {
	return s.getOwned(ctx, tenantID, ownerID, id)
}

func (s *projectService) List(ctx context.Context, tenantID uint64, ownerID string) ([]*types.Project, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, apperrors.NewForbiddenError("a signed-in user is required")
	}
	return s.repo.ListByOwner(ctx, tenantID, ownerID)
}

func (s *projectService) Update(ctx context.Context, tenantID uint64, ownerID, id string, update interfaces.ProjectUpdate) (*types.Project, error) {
	project, err := s.getOwned(ctx, tenantID, ownerID, id)
	if err != nil {
		return nil, err
	}
	name := project.Name
	if update.Name != nil {
		name = *update.Name
	}
	project.Name, err = normalizeProjectName(name)
	if err != nil {
		return nil, err
	}
	if update.Description != nil {
		project.Description = strings.TrimSpace(*update.Description)
	}
	model, editor := project.DefaultModel, project.DefaultEditor
	if update.DefaultModel != nil {
		model = *update.DefaultModel
	}
	if update.DefaultEditor != nil {
		editor = *update.DefaultEditor
	}
	project.DefaultModel, project.DefaultEditor, err = normalizeProjectDefaults(model, editor)
	if err != nil {
		return nil, err
	}
	if update.GitEnabled != nil {
		project.GitEnabled = *update.GitEnabled
	}
	if update.Settings != nil {
		project.Settings = *update.Settings
		if project.Settings == nil {
			project.Settings = types.JSONMap{}
		}
	}
	if err := s.repo.Update(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *projectService) Delete(ctx context.Context, tenantID uint64, ownerID, id string) error {
	if _, err := s.getOwned(ctx, tenantID, ownerID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *projectService) getOwned(ctx context.Context, tenantID uint64, ownerID, id string) (*types.Project, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, apperrors.NewForbiddenError("a signed-in user is required")
	}
	project, err := s.repo.Get(ctx, tenantID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if project == nil || project.OwnerID != ownerID {
		return nil, ErrProjectNotFound
	}
	return project, nil
}

func normalizeProjectName(rawName string) (string, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", apperrors.NewBadRequestError("project name is required")
	}
	if len(name) > projectMaxNameLen {
		return "", apperrors.NewBadRequestError("project name is too long")
	}
	return name, nil
}

func normalizeProjectDefaults(model, editor string) (string, string, error) {
	model, editor = strings.TrimSpace(model), strings.TrimSpace(editor)
	if len(model) > projectMaxDefault || len(editor) > projectMaxDefault {
		return "", "", apperrors.NewBadRequestError("project default is too long")
	}
	return model, editor, nil
}
