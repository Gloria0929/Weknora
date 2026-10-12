package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/gin-gonic/gin"
)

// gitResetRequest names the commit the current branch should be moved back to.
// Repo is the workspace-relative repository root; empty means the workspace root.
type gitResetRequest struct {
	Repo   string `json:"repo"`
	Commit string `json:"commit"`
}

// GitReset moves the current branch back to a local commit with
// `git reset --mixed`: the commits above the target leave the branch, while
// their changes stay in the working tree as uncommitted edits, so nothing the
// user wrote is lost. Published history is never rewritten — the target must be
// the upstream commit or a descendant of it — and the working tree must already
// be clean, so the restored edits are exactly what the dropped commits held.
func (h *ProgrammingHandler) GitReset(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var req gitResetRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid reset request"))
		return
	}
	target := strings.TrimSpace(req.Commit)
	if !validGitHistorySHA(target) {
		c.Error(apperrors.NewBadRequestError("invalid reset commit"))
		return
	}
	if h.workspace == nil {
		c.Error(apperrors.NewConflictError("programming workspace is unavailable"))
		return
	}
	sessionID := programmingSessionID(c)
	layout, store, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}
	workdir, err := h.resolveGitRepoWorkdir(ctx, sessionID, layout, store, req.Repo)
	if err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	refuse := func(detail string) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"reset": false, "detail": detail}})
	}
	// A detached HEAD has no branch to move, and no upstream to protect.
	if _, _, refCode, refErr := h.execGit(ctx, sessionID, workdir, "symbolic-ref --quiet HEAD", 10*time.Second); refErr != nil {
		c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
		return
	} else if refCode != 0 {
		refuse("check out a local branch before reverting")
		return
	}
	headOut, _, headCode, headErr := h.execGit(ctx, sessionID, workdir, "rev-parse --verify HEAD", 10*time.Second)
	if headErr != nil {
		c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
		return
	}
	head := strings.TrimSpace(headOut)
	if headCode != 0 || !validGitHistorySHA(head) {
		refuse("this repository has no commit to move back from")
		return
	}
	if head == target {
		refuse("the branch is already at this commit")
		return
	}
	// Only a commit on the current branch can be a target.
	_, _, ancestorCode, ancestorErr := h.execGit(ctx, sessionID, workdir,
		"merge-base --is-ancestor "+shellQuotePOSIX(target)+" HEAD", 10*time.Second)
	if ancestorErr != nil || ancestorCode > 1 {
		c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
		return
	}
	if ancestorCode != 0 {
		refuse("this commit is not on the current branch")
		return
	}
	// Never rewrite published history: everything the reset drops must still be
	// local, which means the target sits on or above the upstream commit.
	upstreamOut, _, upstreamCode, upstreamErr := h.execGit(ctx, sessionID, workdir,
		"rev-parse --verify --quiet '@{u}'", 10*time.Second)
	if upstreamErr != nil {
		c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
		return
	}
	upstream := strings.TrimSpace(upstreamOut)
	if upstreamCode == 0 && validGitHistorySHA(upstream) && upstream != target {
		_, _, publishCode, publishErr := h.execGit(ctx, sessionID, workdir,
			"merge-base --is-ancestor "+shellQuotePOSIX(upstream)+" "+shellQuotePOSIX(target), 10*time.Second)
		if publishErr != nil || publishCode > 1 {
			c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
			return
		}
		if publishCode != 0 {
			refuse("this commit was already pushed; only unpushed commits can be reverted")
			return
		}
	}
	// A clean tree keeps the restored edits identical to what the dropped
	// commits contained, instead of mixing them with older uncommitted work.
	statusOut, _, statusCode, statusErr := h.execGit(ctx, sessionID, workdir, "status --porcelain=v1 -z", 10*time.Second)
	if statusErr != nil || statusCode != 0 {
		c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
		return
	}
	if len(parseGitStatusPorcelainZ(statusOut)) > 0 {
		refuse("commit or discard your changes before reverting")
		return
	}
	stdout, stderr, resetCode, resetErr := h.execGit(ctx, sessionID, workdir,
		"reset --mixed "+shellQuotePOSIX(target), 30*time.Second)
	if resetErr != nil {
		c.Error(apperrors.NewConflictError("git is unavailable for this workspace"))
		return
	}
	if resetCode != 0 {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = strings.TrimSpace(stdout)
		}
		refuse(detail)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"reset": true}})
}
