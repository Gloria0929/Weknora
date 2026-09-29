package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/stretchr/testify/require"
)

type previewManager struct {
	stubPinnedManager
	stubDesktopProvider
	fakeDesktopShell
	runningErr error
}

func (m *previewManager) RequireRunningSessionSandbox(context.Context, string) error {
	return m.runningErr
}
func (m *previewManager) SessionWorkspaceLayout(context.Context, string) (sandbox.WorkspaceLayout, error) {
	return sandbox.WorkspaceLayout{Root: "/workspace"}, nil
}

func TestWorkspacePreviewURLBoundary(t *testing.T) {
	for _, raw := range []string{"", "http://localhost:5173", "http://127.0.0.1:3000/users?q=a", "http://[::1]:8080/"} {
		require.True(t, validWorkspacePreviewURL(raw), raw)
	}
	for _, raw := range []string{"file:///etc/passwd", "javascript:alert(1)", "http://example.com:3000", "http://localhost", "http://localhost:80", "http://localhost:65536", "http://user@localhost:3000", "http://localhost:49983", "http://localhost:6080", "http://localhost:5900", "http://localhost:9222", "http://localhost.evil:3000"} {
		require.False(t, validWorkspacePreviewURL(raw), raw)
	}
}

func TestWorkspacePreviewUsesPinnedCapabilitiesWithoutExecuting(t *testing.T) {
	mgr := &previewManager{stubDesktopProvider: stubDesktopProvider{advertised: true}}
	resolver := &stubTenantSandboxResolver{mgr: mgr}
	access := NewPinnedSessionSandbox(stubPinReader{configID: "borrowed-config", tenantID: 42}, resolver, nil, nil)
	capability := access.PreviewCapabilities(context.Background(), "session")
	require.True(t, capability.Pinned)
	require.True(t, capability.DesktopEnabled)
	require.Equal(t, "borrowed-config", resolver.lastCfg)
	require.Empty(t, mgr.cmds)
	mgr.advertised = false
	require.False(t, access.PreviewCapabilities(context.Background(), "session").DesktopEnabled)
	var missing *PinnedSessionSandbox
	require.Equal(t, WorkspacePreviewCapabilities{}, missing.PreviewCapabilities(context.Background(), "session"))
}

func TestWorkspacePreviewRefusesPausedAndUnsupportedBeforeExec(t *testing.T) {
	mgr := &previewManager{stubDesktopProvider: stubDesktopProvider{advertised: true}, runningErr: sandbox.ErrSandboxPaused}
	access := NewPinnedSessionSandbox(stubPinReader{configID: "cfg", tenantID: 1}, &stubTenantSandboxResolver{mgr: mgr}, nil, nil)
	_, err := access.OpenWorkspacePreview(context.Background(), "session", WorkspacePreviewRequest{Start: true})
	require.ErrorIs(t, err, sandbox.ErrSandboxPaused)
	require.Empty(t, mgr.cmds)
	mgr.advertised = false
	_, err = access.OpenWorkspacePreview(context.Background(), "session", WorkspacePreviewRequest{})
	require.ErrorIs(t, err, ErrDesktopUnsupported)
	require.Empty(t, mgr.cmds)
}

func TestWorkspacePreviewRunsAsSandboxUserAndParsesResult(t *testing.T) {
	mgr := &previewManager{stubDesktopProvider: stubDesktopProvider{advertised: true}, fakeDesktopShell: fakeDesktopShell{results: []*sandbox.ExecuteResult{{Stdout: `{"status":"ready","url":"http://127.0.0.1:5173/"}`}}}}
	access := NewPinnedSessionSandbox(stubPinReader{configID: "cfg", tenantID: 1}, &stubTenantSandboxResolver{mgr: mgr}, nil, nil)
	result, err := access.OpenWorkspacePreview(context.Background(), "session", WorkspacePreviewRequest{})
	require.NoError(t, err)
	require.Equal(t, "ready", result.Status)
	require.Equal(t, "http://127.0.0.1:5173/", result.URL)
	require.Len(t, mgr.cmds, 1)
	require.False(t, mgr.calls[0].AsRoot)
	require.True(t, mgr.calls[0].SkipWorkspacePrep)
}
