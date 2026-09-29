package handler

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	sessionhandler "github.com/Tencent/WeKnora/internal/handler/session"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// ProjectHandler is the Coding-mode control plane. It owns project metadata
// and creates Coding Threads by reusing the normal Session service. It never
// owns or reimplements an Agent loop.
type ProjectHandler struct {
	projects     interfaces.ProjectService
	folders      interfaces.ProjectFolderService
	sessions     interfaces.SessionService
	workspace    *service.PinnedSessionSandbox
	approvedDirs sessionhandler.HostProjectDirsLoader
}

func NewProjectHandler(projects interfaces.ProjectService, folders interfaces.ProjectFolderService, sessions interfaces.SessionService, workspace *service.PinnedSessionSandbox, approvedDirs sessionhandler.HostProjectDirsLoader) *ProjectHandler {
	return &ProjectHandler{projects: projects, folders: folders, sessions: sessions, workspace: workspace, approvedDirs: approvedDirs}
}

type projectRequest struct {
	Name          *string        `json:"name"`
	Description   *string        `json:"description"`
	DefaultModel  *string        `json:"default_model"`
	DefaultEditor *string        `json:"default_editor"`
	GitEnabled    *bool          `json:"git_enabled"`
	Settings      *types.JSONMap `json:"settings"`
}

type projectFolderRequest struct {
	Name      string `json:"name"`
	LocalPath string `json:"local_path"`
}

type createCodingThreadRequest struct {
	Title string `json:"title"`
}

func (h *ProjectHandler) List(c *gin.Context) {
	projects, err := h.projects.List(c.Request.Context(), projectTenantID(c), projectOwnerID(c))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	out := make([]gin.H, 0, len(projects))
	for _, project := range projects {
		out = append(out, projectResponse(project))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req projectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	project, err := h.projects.Create(c.Request.Context(), projectTenantID(c), projectOwnerID(c), &types.Project{
		Name: valueOrEmpty(req.Name), Description: valueOrEmpty(req.Description),
		DefaultModel: valueOrEmpty(req.DefaultModel), DefaultEditor: valueOrEmpty(req.DefaultEditor), GitEnabled: valueOrFalse(req.GitEnabled),
		Settings: mapOrEmpty(req.Settings),
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": projectResponse(project)})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	project, err := h.projects.Get(c.Request.Context(), projectTenantID(c), projectOwnerID(c), c.Param("project_id"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": projectResponse(project)})
}

func (h *ProjectHandler) Update(c *gin.Context) {
	var req projectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	project, err := h.projects.Update(c.Request.Context(), projectTenantID(c), projectOwnerID(c), c.Param("project_id"), interfaces.ProjectUpdate{
		Name: req.Name, Description: req.Description, DefaultModel: req.DefaultModel,
		DefaultEditor: req.DefaultEditor, GitEnabled: req.GitEnabled, Settings: req.Settings,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": projectResponse(project)})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	projectID := c.Param("project_id")
	threads, err := h.threads(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	if len(threads) > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "请先删除项目中的 Coding 对话，再删除项目"})
		return
	}
	folders, err := h.folders.List(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	for _, folder := range folders {
		if err := h.folders.Delete(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID, folder.ID); err != nil {
			writeProjectError(c, err)
			return
		}
	}
	if err := h.projects.Delete(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID); err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ProjectHandler) ListFolders(c *gin.Context) {
	folders, err := h.folders.List(c.Request.Context(), projectTenantID(c), projectOwnerID(c), c.Param("project_id"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	out := make([]gin.H, 0, len(folders))
	for _, folder := range folders {
		out = append(out, projectFolderResponse(folder))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func (h *ProjectHandler) CreateFolder(c *gin.Context) {
	var req projectFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	folder, err := h.folders.Create(c.Request.Context(), projectTenantID(c), projectOwnerID(c), c.Param("project_id"), &types.ProjectFolder{Name: req.Name, LocalPath: req.LocalPath})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": projectFolderResponse(folder)})
}

func (h *ProjectHandler) DeleteFolder(c *gin.Context) {
	projectID, folderID := c.Param("project_id"), c.Param("folder_id")
	threads, err := h.folderThreads(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID, folderID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	if len(threads) > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "delete Coding Threads before deleting this folder"})
		return
	}
	if err := h.folders.Delete(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID, folderID); err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ProjectHandler) ListFolderThreads(c *gin.Context) {
	threads, err := h.folderThreads(c.Request.Context(), projectTenantID(c), projectOwnerID(c), c.Param("project_id"), c.Param("folder_id"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	out := make([]gin.H, 0, len(threads))
	for _, thread := range threads {
		out = append(out, codingThreadResponse(thread))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

func (h *ProjectHandler) CreateFolderThread(c *gin.Context) {
	var req createCodingThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	projectID := c.Param("project_id")
	folder, err := h.folders.Get(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID, c.Param("folder_id"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	root, ok := sessionhandler.MatchApprovedDir(folder.LocalPath, h.approvedProjectDirs())
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": "project directory is not currently authorized on this local agent"})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = folder.Name
	}
	thread, err := h.sessions.CreateSession(c.Request.Context(), &types.Session{
		TenantID: projectTenantID(c), UserID: projectOwnerID(c), Title: title,
		Mode: types.WorkModeCoding, ProjectID: projectID, ProjectFolderID: folder.ID, HostWorkspaceDir: root,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create coding thread"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": codingThreadResponse(thread)})
}

// ReviewThread returns the current, read-only Git working-tree diff for one
// coding thread. Agent edits are live changes in the authorized directory, so
// this endpoint deliberately does not pretend there is a separate patch to
// accept or reject. It runs a fixed Git command through the same session
// workspace boundary used by the Agent and terminal.
func (h *ProjectHandler) ReviewThread(c *gin.Context) {
	projectID := c.Param("project_id")
	folderID := c.Param("folder_id")
	threadID := strings.TrimSpace(c.Param("thread_id"))
	threads, err := h.threads(c.Request.Context(), projectTenantID(c), projectOwnerID(c), projectID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	var thread *types.Session
	for _, candidate := range threads {
		if candidate.ID == threadID {
			thread = candidate
			break
		}
	}
	if thread == nil || thread.ProjectFolderID != folderID {
		c.JSON(http.StatusNotFound, gin.H{"error": "coding thread not found"})
		return
	}
	if h.workspace == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "coding workspace is unavailable"})
		return
	}
	layout, err := h.workspace.SessionWorkspaceLayout(c.Request.Context(), thread.ID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "workspace is not running; send a Coding task first"})
		return
	}
	result, err := h.workspace.ExecWorkspaceCommand(
		c.Request.Context(), thread.ID, "git diff --no-ext-diff --no-color -- .", layout.Root, 15*time.Second, nil,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Git review is unavailable for this workspace"})
		return
	}
	if result.ExitCode != 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"available": false, "diff": "", "message": strings.TrimSpace(result.Stderr),
		}})
		return
	}
	diff := result.Stdout
	const maxDiffChars = 512 * 1024
	truncated := len(diff) > maxDiffChars
	if truncated {
		diff = diff[:maxDiffChars]
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"available": true, "diff": diff, "truncated": truncated,
		"message": "Diff reflects the live authorized working tree.",
	}})
}

func (h *ProjectHandler) threads(ctx context.Context, tenantID uint64, ownerID, projectID string) ([]*types.Session, error) {
	if _, err := h.projects.Get(ctx, tenantID, ownerID, projectID); err != nil {
		return nil, err
	}
	sessions, err := h.sessions.GetSessionsByTenant(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*types.Session, 0)
	for _, thread := range sessions {
		if thread.Mode == types.WorkModeCoding && thread.ProjectID == projectID {
			out = append(out, thread)
		}
	}
	return out, nil
}

func (h *ProjectHandler) folderThreads(ctx context.Context, tenantID uint64, ownerID, projectID, folderID string) ([]*types.Session, error) {
	if _, err := h.folders.Get(ctx, tenantID, ownerID, projectID, folderID); err != nil {
		return nil, err
	}
	threads, err := h.threads(ctx, tenantID, ownerID, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]*types.Session, 0)
	for _, thread := range threads {
		if thread.ProjectFolderID == folderID {
			out = append(out, thread)
		}
	}
	return out, nil
}

func (h *ProjectHandler) approvedProjectDirs() []string {
	if h.approvedDirs == nil {
		return nil
	}
	return h.approvedDirs()
}

func projectTenantID(c *gin.Context) uint64 { return c.GetUint64(types.TenantIDContextKey.String()) }
func projectOwnerID(c *gin.Context) string {
	return types.CallerFromContext(c.Request.Context()).UserID
}
func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func valueOrFalse(v *bool) bool { return v != nil && *v }
func mapOrEmpty(v *types.JSONMap) types.JSONMap {
	if v == nil {
		return types.JSONMap{}
	}
	return *v
}

func projectResponse(project *types.Project) gin.H {
	return gin.H{"id": project.ID, "tenant_id": project.TenantID, "name": project.Name,
		"description": project.Description, "default_model": project.DefaultModel, "default_editor": project.DefaultEditor,
		"git_enabled": project.GitEnabled, "settings": project.Settings,
		"created_at": project.CreatedAt, "updated_at": project.UpdatedAt}
}

func projectFolderResponse(folder *types.ProjectFolder) gin.H {
	return gin.H{"id": folder.ID, "project_id": folder.ProjectID, "name": folder.Name, "local_path": folder.LocalPath,
		"created_at": folder.CreatedAt, "updated_at": folder.UpdatedAt}
}

func codingThreadResponse(thread *types.Session) gin.H {
	return gin.H{"id": thread.ID, "project_id": thread.ProjectID, "folder_id": thread.ProjectFolderID, "mode": thread.Mode, "title": thread.Title,
		"created_at": thread.CreatedAt, "updated_at": thread.UpdatedAt}
}

func writeProjectError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, service.ErrProjectNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		status := http.StatusBadRequest
		switch appErr.Code {
		case apperrors.ErrNotFound:
			status = http.StatusNotFound
		case apperrors.ErrForbidden, apperrors.ErrUnauthorized:
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "project operation failed"})
}
