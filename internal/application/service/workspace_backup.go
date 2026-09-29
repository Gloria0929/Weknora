package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/sandbox"
)

//go:embed workspace_backup.py
var workspaceBackupScript string

// BackupSource stages a source archive for the artifact collector. Unlike a
// git checkpoint inside the guest, the collected archive survives destruction
// of that guest. This runs only against the turn's existing remote sandbox.
func (c *WorkspaceCheckpointer) BackupSource(ctx context.Context, sessionID, sandboxID, messageID, outputDir string, maxBytes int64) {
	if c == nil || c.runner == nil || strings.TrimSpace(sessionID) == "" || strings.TrimSpace(sandboxID) == "" || strings.TrimSpace(messageID) == "" {
		return
	}
	if v, ok := c.runner.(WorkspaceVersioning); ok && !v.VersionsWorkspace(ctx, sessionID) {
		return
	}
	outputDir, valid := sandbox.ValidatedSessionOutputDir(outputDir)
	runner, supported := c.runner.(sandbox.SessionInstallShellExecutor)
	if !valid || !supported || maxBytes <= 0 {
		return
	}
	args, _ := json.Marshal(struct {
		Root     string `json:"root"`
		Output   string `json:"output"`
		MaxBytes int64  `json:"max_bytes"`
		Turn     string `json:"turn"`
	}{sandbox.SessionWorkspaceRoot, outputDir, maxBytes, messageID})
	const timeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := "python3 -c " + sandbox.ShellQuote(workspaceBackupScript) + " " + sandbox.ShellQuote(string(args))
	result, err := runner.ExecShellCommandWithOptions(ctx, sessionID, command, sandbox.ShellExecOptions{
		WorkDir: sandbox.SessionWorkspaceRoot, Timeout: timeout,
		SkipWorkspacePrep: true, ExpectedSandboxID: sandboxID,
	})
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceBackup] exec failed session=%s: %v", sessionID, err)
	} else if result == nil || result.ExitCode != 0 {
		detail := "no result"
		if result != nil {
			detail = truncateForLog(result.Stderr)
		}
		logger.Warnf(ctx, "[WorkspaceBackup] archive not updated session=%s: %s", sessionID, detail)
	}
}
