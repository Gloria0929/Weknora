package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// workspaceStateLive and workspaceStateExpired tell the code panel whether the
// session still owns a sandbox. A reclaimed sandbox leaves the tree empty, and
// an empty tree on its own reads as "nothing was created yet".
const (
	workspaceStateLive     = "live"
	workspaceStateExpired  = "expired"
	workspaceExpiredReason = "workspace_expired"
)

const (
	programmingMaxFileBytes = 2 * 1024 * 1024
	programmingMaxCommand   = 8 * 1024
	programmingMaxTimeout   = 60 * time.Second
)

// ProgrammingHandler exposes the P0 programming-space surface on top of the
// existing session sandbox. A session is the current compatibility boundary:
// it already owns the Agent conversation, sandbox pin, user scope and
// terminal lifecycle, so the programming UI cannot accidentally escape those
// existing authorization rules.
type ProgrammingHandler struct {
	previewLeases *previewLeaseStore
	sessions      interfaces.SessionService
	workspace     *service.PinnedSessionSandbox
}

func NewProgrammingHandler(
	sessions interfaces.SessionService,
	workspace *service.PinnedSessionSandbox,
) *ProgrammingHandler {
	return &ProgrammingHandler{sessions: sessions, workspace: workspace}
}

type programmingFileRequest struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	BaseHash string `json:"base_hash,omitempty"`
}

type programmingMoveFileRequest struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type programmingCommandRequest struct {
	Command string `json:"command"`
	Path    string `json:"path,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type programmingNode struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Size     int64     `json:"size,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

type programmingFile struct {
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	Hash     string    `json:"hash"`
	Size     int       `json:"size"`
	Modified time.Time `json:"modified,omitempty"`
}

// ListTree returns the current workspace files. The store performs the
// provider-specific path checks; this handler only projects absolute provider
// paths back into the relative paths shown in the editor.
func (h *ProgrammingHandler) ListTree(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	sessionID := programmingSessionID(c)
	requestedPath := strings.TrimSpace(strings.ReplaceAll(c.Query("path"), "\\", "/"))
	directory := layout.Root
	if requestedPath != "" {
		resolved, rel, resolveErr := resolveWorkspacePath(layout, requestedPath)
		if resolveErr != nil || rel == "" {
			c.Error(apperrors.NewBadRequestError("path must stay inside the workspace"))
			return
		}
		directory = resolved
	}

	var (
		entries []sandbox.RemoteDirEntry
		err     error
	)
	if lister, supported := store.(interface {
		ListSessionDirectory(context.Context, string, string) ([]sandbox.RemoteDirEntry, error)
	}); supported {
		entries, err = lister.ListSessionDirectory(ctx, sessionID, directory)
	} else {
		entries, err = store.ListSessionFiles(ctx, sessionID, directory)
	}
	workspaceState := workspaceStateLive
	if errors.Is(err, sandbox.ErrNoLiveSessionSandbox) {
		// No live sandbox: an empty tree either way. A session that still
		// carries a binding had its sandbox reaped, and the panel should say
		// so; a session that was never provisioned is simply empty.
		if h.reclaimedWorkspace(ctx, sessionID) {
			workspaceState = workspaceStateExpired
		}
		entries, err = nil, nil
	}
	if err != nil {
		c.Error(apperrors.NewConflictError("workspace is not running or cannot be listed"))
		return
	}

	nodes := make([]programmingNode, 0, len(entries)*2)
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		rel, err := relativeWorkspacePath(layout.Root, entry.Path)
		if err != nil || rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		nodes = append(nodes, programmingNode{
			Path:     rel,
			Name:     entry.Name,
			Kind:     map[bool]string{true: "directory", false: "file"}[entry.Type == sandbox.RemoteEntryDir],
			Size:     entry.Size,
			Modified: entry.ModTime,
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind == "directory"
		}
		return nodes[i].Path < nodes[j].Path
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"root":            layout.Root,
		"origin":          map[bool]string{true: "host", false: "sandbox"}[layout.IsHost()],
		"workspace_state": workspaceState,
		"nodes":           nodes,
	}})
}

// workspaceExpiredError converts a reclaimed-sandbox failure into the
// answer the code panel acts on. HTTP 404 keeps the client's "this path is
// gone" handling — a reclaimed workspace really does not contain the file —
// while the reason tells it not to blame the path.
func workspaceExpiredError() *apperrors.AppError {
	return apperrors.NewNotFoundError(
		"the session workspace has been reclaimed, so its files are no longer available").
		WithDetails(gin.H{"reason": workspaceExpiredReason})
}

// reclaimedWorkspace reports whether the session's file operations failed
// because the sandbox was reaped. A session that never had a sandbox has no
// binding to lose, and its empty workspace is normal rather than expired.
func (h *ProgrammingHandler) reclaimedWorkspace(ctx context.Context, sessionID string) bool {
	if h == nil || h.workspace == nil {
		return false
	}
	_, bound := h.workspace.BoundSandboxID(ctx, sessionID)
	return bound
}

// reclaimedWorkspaceError turns a no-live-sandbox failure into the 404 that
// explains a reaped sandbox, or returns nil for any other failure.
func (h *ProgrammingHandler) reclaimedWorkspaceError(
	ctx context.Context, sessionID string, err error,
) error {
	if !errors.Is(err, sandbox.ErrNoLiveSessionSandbox) || !h.reclaimedWorkspace(ctx, sessionID) {
		return nil
	}
	return workspaceExpiredError()
}

// workspaceWriteConflict answers a failed mutation. A reclaimed sandbox is a
// conflict rather than a bad request, but the reason lets the panel offer the
// one action that helps: ask the Agent to rebuild the workspace.
func (h *ProgrammingHandler) workspaceWriteConflict(
	c *gin.Context, sessionID string, err error, message string,
) {
	if errors.Is(err, sandbox.ErrNoLiveSessionSandbox) &&
		h.reclaimedWorkspace(c.Request.Context(), sessionID) {
		c.Error(apperrors.NewConflictError(
			"the session workspace has been reclaimed and cannot be modified").
			WithDetails(gin.H{"reason": workspaceExpiredReason}))
		return
	}
	c.Error(apperrors.NewConflictError(message))
}

// ReadFile reads one text file after resolving it under the active workspace
// root. Binary files and oversized files are rejected instead of being
// streamed into the editor.
func (h *ProgrammingHandler) ReadFile(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	absolute, rel, err := resolveWorkspacePath(layout, c.Query("path"))
	if err != nil || rel == "" {
		c.Error(apperrors.NewBadRequestError("path must stay inside the workspace"))
		return
	}
	sessionID := programmingSessionID(c)
	stat, err := store.StatSessionFile(ctx, sessionID, absolute)
	if err != nil {
		// A reclaimed sandbox is reported before plain not-found: both answer
		// 404, but only the expired case can tell the reader that the whole
		// workspace is gone instead of sending them looking for a path bug.
		if expired := h.reclaimedWorkspaceError(ctx, sessionID, err); expired != nil {
			c.Error(expired)
			return
		}
		// Host workspaces surface the operating system's own missing-file
		// error, which means the same thing as a provider not-found.
		if sandbox.IsRemoteNotFound(err) || errors.Is(err, fs.ErrNotExist) {
			c.Error(apperrors.NewNotFoundError("file not found"))
			return
		}
		c.Error(apperrors.NewConflictError("workspace is not running or the file cannot be read"))
		return
	}
	if stat == nil || stat.Type != sandbox.RemoteEntryFile {
		c.Error(apperrors.NewNotFoundError("file not found"))
		return
	}
	if stat.Size > programmingMaxFileBytes {
		c.Error(apperrors.NewBadRequestError("file is too large for the editor"))
		return
	}
	content, err := store.ReadSessionFile(ctx, sessionID, absolute)
	if err != nil {
		if expired := h.reclaimedWorkspaceError(ctx, sessionID, err); expired != nil {
			c.Error(expired)
			return
		}
		if sandbox.IsRemoteNotFound(err) || errors.Is(err, fs.ErrNotExist) {
			c.Error(apperrors.NewNotFoundError("file not found"))
			return
		}
		c.Error(apperrors.NewConflictError("workspace is not running or the file cannot be read"))
		return
	}
	if strings.IndexByte(string(content), 0) >= 0 {
		c.Error(apperrors.NewBadRequestError("binary files cannot be opened in the editor"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": filePayload(rel, content, stat.ModTime)})
}

// WriteFile applies an optimistic hash check before writing. This is the
// first version of the requested FileVersion contract: clients send the hash
// they opened, and an out-of-date write receives VERSION_CONFLICT instead of
// overwriting another editor or Agent's change.
func (h *ProgrammingHandler) WriteFile(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var req programmingFileRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid file request"))
		return
	}
	if len(req.Content) > programmingMaxFileBytes {
		c.Error(apperrors.NewBadRequestError("file is too large"))
		return
	}
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	absolute, rel, err := resolveWorkspacePath(layout, req.Path)
	if err != nil || rel == "" {
		c.Error(apperrors.NewBadRequestError("path must stay inside the workspace"))
		return
	}
	sessionID := programmingSessionID(c)
	current, stat, readErr := readWorkspaceFile(ctx, store, sessionID, absolute)
	if readErr == nil && req.BaseHash != "" && req.BaseHash != contentHash(current) {
		c.JSON(http.StatusConflict, gin.H{"success": false, "code": "VERSION_CONFLICT", "message": "file changed since it was opened", "data": filePayload(rel, current, stat.ModTime)})
		return
	}
	if readErr != nil && req.BaseHash != "" {
		c.JSON(http.StatusConflict, gin.H{"success": false, "code": "VERSION_CONFLICT", "message": "file was created or removed since it was opened"})
		return
	}
	if err := store.WriteSessionWorkspaceFile(ctx, sessionID, absolute, []byte(req.Content)); err != nil {
		h.workspaceWriteConflict(c, sessionID, err, "workspace is not writable")
		return
	}
	now := time.Now()
	if latest, latestStat, err := readWorkspaceFile(ctx, store, sessionID, absolute); err == nil {
		current, now = latest, latestStat.ModTime
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": filePayload(rel, current, now)})
}

// CreateFile creates an empty text file and refuses to overwrite an existing
// file. The editor uses this endpoint for the explorer's new-file action so a
// stale, partially loaded tree cannot accidentally replace a project file.
func (h *ProgrammingHandler) CreateFile(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var req programmingFileRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid file request"))
		return
	}
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	absolute, rel, err := resolveWorkspacePath(layout, req.Path)
	if err != nil || rel == "" {
		c.Error(apperrors.NewBadRequestError("path must stay inside the workspace"))
		return
	}
	sessionID := programmingSessionID(c)
	if stat, statErr := store.StatSessionFile(ctx, sessionID, absolute); statErr == nil && stat != nil {
		c.Error(apperrors.NewConflictError("file already exists"))
		return
	}
	if err := store.WriteSessionWorkspaceFile(ctx, sessionID, absolute, []byte{}); err != nil {
		h.workspaceWriteConflict(c, sessionID, err, "workspace is not writable")
		return
	}
	now := time.Now()
	if stat, statErr := store.StatSessionFile(ctx, sessionID, absolute); statErr == nil && stat != nil {
		now = stat.ModTime
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": filePayload(rel, nil, now)})
}

// MoveFile moves one regular file to another existing directory. The provider
// keeps its filesystem-specific move implementation behind the file store.
func (h *ProgrammingHandler) MoveFile(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var req programmingMoveFileRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid move request"))
		return
	}
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	source, sourceRel, err := resolveWorkspacePath(layout, req.Source)
	if err != nil || sourceRel == "" {
		c.Error(apperrors.NewBadRequestError("source path must stay inside the workspace"))
		return
	}
	target, targetRel, err := resolveWorkspacePath(layout, req.Target)
	if err != nil || targetRel == "" || path.Clean(source) == path.Clean(target) {
		c.Error(apperrors.NewBadRequestError("target path must stay inside the workspace"))
		return
	}
	sessionID := programmingSessionID(c)
	sourceStat, statErr := store.StatSessionFile(ctx, sessionID, source)
	if expired := h.reclaimedWorkspaceError(ctx, sessionID, statErr); expired != nil {
		c.Error(expired)
		return
	}
	if statErr != nil || sourceStat == nil || sourceStat.Type != sandbox.RemoteEntryFile {
		c.Error(apperrors.NewNotFoundError("source file not found"))
		return
	}
	if targetStat, targetErr := store.StatSessionFile(ctx, sessionID, target); targetErr == nil && targetStat != nil {
		c.Error(apperrors.NewConflictError("target file already exists"))
		return
	}
	parentStat, parentErr := store.StatSessionFile(ctx, sessionID, path.Dir(target))
	if parentErr != nil || parentStat == nil || parentStat.Type != sandbox.RemoteEntryDir {
		c.Error(apperrors.NewBadRequestError("target directory does not exist"))
		return
	}
	mover, supported := store.(interface {
		MoveSessionWorkspaceFile(context.Context, string, string, string) error
	})
	if !supported {
		c.Error(apperrors.NewConflictError("workspace does not support moving files"))
		return
	}
	if err := mover.MoveSessionWorkspaceFile(ctx, sessionID, source, target); err != nil {
		h.workspaceWriteConflict(c, sessionID, err, "workspace could not move the file")
		return
	}
	payload := filePayload(targetRel, nil, sourceStat.ModTime)
	payload.Size = int(sourceStat.Size)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": payload})
}

// DeleteFile removes one regular file from the active workspace.
func (h *ProgrammingHandler) DeleteFile(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var req programmingFileRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid delete request"))
		return
	}
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	absolute, rel, err := resolveWorkspacePath(layout, req.Path)
	if err != nil || rel == "" {
		c.Error(apperrors.NewBadRequestError("path must stay inside the workspace"))
		return
	}
	sessionID := programmingSessionID(c)
	stat, statErr := store.StatSessionFile(ctx, sessionID, absolute)
	if expired := h.reclaimedWorkspaceError(ctx, sessionID, statErr); expired != nil {
		c.Error(expired)
		return
	}
	if statErr != nil || stat == nil || stat.Type != sandbox.RemoteEntryFile {
		c.Error(apperrors.NewNotFoundError("file not found"))
		return
	}
	remover, supported := store.(interface {
		RemoveSessionWorkspaceFile(context.Context, string, string) error
	})
	if !supported {
		c.Error(apperrors.NewConflictError("workspace does not support deleting files"))
		return
	}
	if err := remover.RemoveSessionWorkspaceFile(ctx, sessionID, absolute); err != nil {
		h.workspaceWriteConflict(c, sessionID, err, "workspace file could not be deleted")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"path": rel}})
}

// RunCommand powers the first terminal/output panel. It shares the same
// workspace manager and path guard as file access; the underlying sandbox
// backend remains responsible for command policy and OS isolation.
func (h *ProgrammingHandler) RunCommand(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var req programmingCommandRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil || strings.TrimSpace(req.Command) == "" {
		c.Error(apperrors.NewBadRequestError("command is required"))
		return
	}
	if len(req.Command) > programmingMaxCommand {
		c.Error(apperrors.NewBadRequestError("command is too long"))
		return
	}
	layout, _, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	workDir := layout.Root
	if strings.TrimSpace(req.Path) != "" {
		var err error
		workDir, _, err = resolveWorkspacePath(layout, req.Path)
		if err != nil {
			c.Error(apperrors.NewBadRequestError("working directory must stay inside the workspace"))
			return
		}
	}
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}
	if timeout > programmingMaxTimeout {
		timeout = programmingMaxTimeout
	}
	result, err := h.workspace.ExecWorkspaceCommand(ctx, programmingSessionID(c), req.Command, workDir, timeout, nil)
	if err != nil {
		c.Error(apperrors.NewConflictError("workspace command is unavailable"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"stdout":      result.Stdout,
		"stderr":      result.Stderr,
		"exit_code":   result.ExitCode,
		"duration_ms": result.Duration.Milliseconds(),
		"killed":      result.Killed,
	}})
}

func (h *ProgrammingHandler) loadOwnedSession(c *gin.Context) (*types.Session, bool) {
	if h == nil || h.sessions == nil {
		c.Error(apperrors.NewInternalServerError("programming service is unavailable"))
		return nil, false
	}
	session, err := h.sessions.GetOwnedSession(c.Request.Context(), programmingSessionID(c))
	if err != nil || session == nil {
		c.Error(apperrors.NewNotFoundError("session not found"))
		return nil, false
	}
	return session, true
}

func (h *ProgrammingHandler) resolveWorkspace(c *gin.Context) (sandbox.WorkspaceLayout, sandbox.SessionFileStore, bool) {
	if h == nil || h.workspace == nil {
		c.Error(apperrors.NewConflictError("programming workspace is unavailable"))
		return sandbox.WorkspaceLayout{}, nil, false
	}
	ctx := c.Request.Context()
	sessionID := programmingSessionID(c)
	layout, err := h.workspace.SessionWorkspaceLayout(ctx, sessionID)
	if err != nil {
		c.Error(apperrors.NewConflictError("workspace is not running"))
		return sandbox.WorkspaceLayout{}, nil, false
	}
	store, err := h.workspace.SessionFileStore(ctx, sessionID)
	if err != nil {
		c.Error(apperrors.NewConflictError("workspace file access is unavailable"))
		return sandbox.WorkspaceLayout{}, nil, false
	}
	return layout, store, true
}

func programmingSessionID(c *gin.Context) string {
	if id := strings.TrimSpace(c.Param("session_id")); id != "" {
		return id
	}
	return strings.TrimSpace(c.Param("id"))
}

func resolveWorkspacePath(layout sandbox.WorkspaceLayout, raw string) (string, string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" || strings.ContainsRune(raw, '\x00') {
		return "", "", fmt.Errorf("path is required")
	}
	root := path.Clean(layout.Root)
	candidate := raw
	if !strings.HasPrefix(candidate, "/") {
		candidate = path.Join(root, candidate)
	} else {
		candidate = path.Clean(candidate)
	}
	if candidate != root && !strings.HasPrefix(candidate, root+"/") {
		return "", "", fmt.Errorf("path escaped workspace")
	}
	rel := strings.TrimPrefix(candidate, root)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "", fmt.Errorf("invalid workspace path")
	}
	return candidate, rel, nil
}

func relativeWorkspacePath(root, absolute string) (string, error) {
	root = path.Clean(root)
	absolute = path.Clean(strings.ReplaceAll(absolute, "\\", "/"))
	if absolute == root {
		return "", nil
	}
	if !strings.HasPrefix(absolute, root+"/") {
		return "", fmt.Errorf("path outside workspace")
	}
	return strings.TrimPrefix(absolute, root+"/"), nil
}

func readWorkspaceFile(ctx context.Context, store sandbox.SessionFileStore, sessionID, filePath string) ([]byte, *sandbox.RemoteStatEntry, error) {
	stat, err := store.StatSessionFile(ctx, sessionID, filePath)
	if err != nil || stat.Type != sandbox.RemoteEntryFile || stat.Size > programmingMaxFileBytes {
		return nil, nil, fmt.Errorf("file is unavailable")
	}
	content, err := store.ReadSessionFile(ctx, sessionID, filePath)
	if err != nil {
		return nil, nil, err
	}
	return content, stat, nil
}

func filePayload(rel string, content []byte, modified time.Time) programmingFile {
	return programmingFile{
		Path: rel, Content: string(content), Hash: contentHash(content), Size: len(content), Modified: modified,
	}
}

func contentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
