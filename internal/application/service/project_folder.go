package service

import (
	"context"
	"path/filepath"
	"strings"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// ProjectFolderService keeps the Codex-style project/folder hierarchy
// separate from knowledge bases. The stored path remains an authorization
// candidate only; thread creation performs the actual host-directory check.
type projectFolderService struct {
	projects interfaces.ProjectService
	repo     interfaces.ProjectFolderRepository
}

func NewProjectFolderService(projects interfaces.ProjectService, repo interfaces.ProjectFolderRepository) interfaces.ProjectFolderService {
	return &projectFolderService{projects: projects, repo: repo}
}

func (s *projectFolderService) Create(ctx context.Context, tenantID uint64, ownerID, projectID string, in *types.ProjectFolder) (*types.ProjectFolder, error) {
	if in == nil {
		return nil, apperrors.NewBadRequestError("folder is required")
	}
	if _, err := s.projects.Get(ctx, tenantID, ownerID, projectID); err != nil {
		return nil, err
	}
	name, localPath, err := normalizeProjectFolder(in.Name, in.LocalPath)
	if err != nil {
		return nil, err
	}
	folder := &types.ProjectFolder{ProjectID: projectID, TenantID: tenantID, OwnerID: ownerID, Name: name, LocalPath: localPath}
	if err := s.repo.Create(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *projectFolderService) Get(ctx context.Context, tenantID uint64, ownerID, projectID, folderID string) (*types.ProjectFolder, error) {
	if _, err := s.projects.Get(ctx, tenantID, ownerID, projectID); err != nil {
		return nil, err
	}
	folder, err := s.repo.Get(ctx, tenantID, strings.TrimSpace(folderID))
	if err != nil {
		return nil, err
	}
	if folder == nil || folder.ProjectID != projectID || folder.OwnerID != ownerID {
		return nil, ErrProjectNotFound
	}
	return folder, nil
}

func (s *projectFolderService) List(ctx context.Context, tenantID uint64, ownerID, projectID string) ([]*types.ProjectFolder, error) {
	if _, err := s.projects.Get(ctx, tenantID, ownerID, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, tenantID, projectID, ownerID)
}

func (s *projectFolderService) Delete(ctx context.Context, tenantID uint64, ownerID, projectID, folderID string) error {
	folder, err := s.Get(ctx, tenantID, ownerID, projectID, folderID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, folder.ID)
}

func normalizeProjectFolder(rawName, rawPath string) (string, string, error) {
	localPath := strings.TrimSpace(rawPath)
	if localPath == "" || !filepath.IsAbs(localPath) {
		return "", "", apperrors.NewBadRequestError("folder local_path must be an absolute path")
	}
	localPath = filepath.Clean(localPath)
	if len(localPath) > projectMaxPathLen {
		return "", "", apperrors.NewBadRequestError("folder local_path is too long")
	}
	name := strings.TrimSpace(rawName)
	if name == "" {
		name = filepath.Base(localPath)
	}
	if len(name) > projectMaxNameLen {
		return "", "", apperrors.NewBadRequestError("folder name is too long")
	}
	return name, localPath, nil
}
