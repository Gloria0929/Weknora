package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/sandbox"
)

//go:embed workspace_preview.py
var workspacePreviewScript string

type WorkspacePreviewCapabilities struct {
	Pinned         bool `json:"pinned"`
	DesktopEnabled bool `json:"desktop_enabled"`
}

// The session pin is authoritative, including for borrowed agents. This does
// not connect, create or resume a sandbox, or disclose its config credentials.
func (a *PinnedSessionSandbox) PreviewCapabilities(ctx context.Context, sessionID string) WorkspacePreviewCapabilities {
	mgr := a.manager(ctx, sessionID)
	return WorkspacePreviewCapabilities{Pinned: mgr != nil, DesktopEnabled: mgr != nil && requireDesktopCapable(mgr) == nil}
}

type WorkspacePreviewRequest struct {
	URL      string `json:"url"`
	Start    bool   `json:"start"`
	Action   string `json:"action,omitempty"`
	Viewport string `json:"viewport,omitempty"`
}

type WorkspacePreviewResult struct {
	Status   string `json:"status"`
	URL      string `json:"url,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Viewport string `json:"viewport,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

// Only guest loopback web servers belong in a project preview. In particular,
// never turn the server into a URL proxy or expose envd / desktop control ports.
func validWorkspacePreviewURL(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Scheme != "http" {
		return false
	}
	if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	return err == nil && port >= 1024 && port <= 65535 && port != 49983 && port != 5900 && port != 6080 && port != 9222
}

// OpenWorkspacePreview uses the browser inside the already-running session.
// Its pixels travel through the existing authenticated desktop relay; no app
// origin, sandbox token, or unauthenticated development port reaches the host UI.
func (a *PinnedSessionSandbox) OpenWorkspacePreview(ctx context.Context, sessionID string, request WorkspacePreviewRequest) (*WorkspacePreviewResult, error) {
	request.URL = strings.TrimSpace(request.URL)
	if request.Action != "" && request.Action != "resize" && request.Action != "reload" {
		return nil, fmt.Errorf("invalid preview action")
	}
	if request.Viewport != "" && request.Viewport != "desktop" && request.Viewport != "mobile" {
		return nil, fmt.Errorf("invalid preview viewport")
	}
	if !validWorkspacePreviewURL(request.URL) {
		return nil, fmt.Errorf("preview URL must be a sandbox localhost HTTP URL with a non-reserved port")
	}
	mgr := a.manager(ctx, sessionID)
	if err := requireDesktopCapable(mgr); err != nil {
		return nil, err
	}
	if err := requireRunningSessionSandbox(ctx, mgr, sessionID); err != nil {
		return nil, err
	}
	runner, ok := mgr.(sessionShellOptionsExecutor)
	if !ok {
		return nil, ErrDesktopUnsupported
	}
	provider, ok := mgr.(sandbox.SessionWorkspaceLayoutProvider)
	if !ok {
		return nil, ErrDesktopUnsupported
	}
	layout, err := provider.SessionWorkspaceLayout(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	layout = layout.Normalized()
	if !layout.HasRoot() || layout.IsHost() {
		return nil, fmt.Errorf("sandbox workspace root is unavailable")
	}
	args, _ := json.Marshal(struct {
		Root string `json:"root"`
		WorkspacePreviewRequest
	}{layout.Root, request})
	command := "python3 -c " + sandbox.ShellQuote(workspacePreviewScript) + " " + sandbox.ShellQuote(string(args))
	result, err := runner.ExecShellCommandWithOptions(ctx, sessionID, command, sandbox.ShellExecOptions{
		Timeout: 45 * time.Second, SkipWorkspacePrep: true,
	})
	if err != nil {
		return nil, err
	}
	if result == nil || !result.IsSuccess() {
		return nil, fmt.Errorf("sandbox preview command failed")
	}
	var preview WorkspacePreviewResult
	if err := json.Unmarshal([]byte(result.Stdout), &preview); err != nil {
		return nil, fmt.Errorf("invalid sandbox preview response")
	}
	return &preview, nil
}
