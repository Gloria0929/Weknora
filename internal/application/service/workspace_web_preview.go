package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/sandbox"
)

//go:embed workspace_web_preview.py
var workspaceWebPreviewScript string

//go:embed workspace_web_relay.py
var workspaceWebRelayScript string

type WorkspaceWebPreview struct {
	Status    string `json:"status"`
	URL       string `json:"url,omitempty"`
	Detail    string `json:"detail,omitempty"`
	Port      int    `json:"port,omitempty"`
	RelayPort int    `json:"relay_port,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

func (a *PinnedSessionSandbox) DiscoverWebPreview(ctx context.Context, sessionID string, start bool) (*WorkspaceWebPreview, error) {
	mgr := a.manager(ctx, sessionID)
	if mgr == nil {
		return nil, sandbox.ErrNoLiveSessionSandbox
	}
	if err := requireRunningSessionSandbox(ctx, mgr, sessionID); err != nil {
		return nil, err
	}
	runner, ok := mgr.(sessionShellOptionsExecutor)
	if !ok {
		return nil, sandbox.ErrPreviewUnsupported
	}
	if _, ok := mgr.(sandbox.SessionPreviewTransport); !ok {
		return nil, sandbox.ErrPreviewUnsupported
	}
	provider, ok := mgr.(sandbox.SessionWorkspaceLayoutProvider)
	if !ok {
		return nil, sandbox.ErrPreviewUnsupported
	}
	layout, err := provider.SessionWorkspaceLayout(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	layout = layout.Normalized()
	if !layout.HasRoot() || layout.IsHost() {
		return nil, sandbox.ErrPreviewUnsupported
	}
	args, _ := json.Marshal(map[string]any{"root": layout.Root, "start": start, "relay_script": workspaceWebRelayScript})
	helpers := strings.Split(workspacePreviewScript, "if __name__ == \"__main__\":")[0]
	command := "python3 -c " + sandbox.ShellQuote(helpers+"\n"+workspaceWebPreviewScript) + " " + sandbox.ShellQuote(string(args))
	result, err := runner.ExecShellCommandWithOptions(ctx, sessionID, command, sandbox.ShellExecOptions{Timeout: 40 * time.Second, SkipWorkspacePrep: true})
	if err != nil {
		return nil, err
	}
	if result == nil || !result.IsSuccess() {
		return nil, fmt.Errorf("project discovery failed")
	}
	var preview WorkspaceWebPreview
	if err := json.Unmarshal([]byte(result.Stdout), &preview); err != nil {
		return nil, fmt.Errorf("invalid project discovery response")
	}
	if preview.Status == "ready" && (!sandbox.ValidProjectPort(preview.Port) || !sandbox.ValidProjectPort(preview.RelayPort) || len(preview.Secret) != 64) {
		return nil, fmt.Errorf("invalid project relay")
	}
	return &preview, nil
}

func (a *PinnedSessionSandbox) RoundTripWebPreview(ctx context.Context, sessionID, sandboxID string, port int, req *http.Request) (*http.Response, error) {
	mgr := a.manager(ctx, sessionID)
	if transport, ok := mgr.(sandbox.SessionPreviewTransport); ok {
		return transport.RoundTripSessionPreview(ctx, sessionID, sandboxID, port, req)
	}
	return nil, sandbox.ErrPreviewUnsupported
}
