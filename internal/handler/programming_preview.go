package handler

import (
	"errors"
	"net/http"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/gin-gonic/gin"
)

func (h *ProgrammingHandler) PreviewCapabilities(c *gin.Context) {
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.workspace.PreviewCapabilities(c.Request.Context(), programmingSessionID(c))})
}

func (h *ProgrammingHandler) OpenPreview(c *gin.Context) {
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var request service.WorkspacePreviewRequest
	if err := c.ShouldBindJSON(&request); err != nil || len(request.URL) > 2048 {
		c.Error(apperrors.NewBadRequestError("invalid preview request"))
		return
	}
	result, err := h.workspace.OpenWorkspacePreview(c.Request.Context(), programmingSessionID(c), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDesktopUnsupported):
			c.JSON(http.StatusOK, gin.H{"success": true, "data": service.WorkspacePreviewResult{Status: "unsupported"}})
		case errors.Is(err, sandbox.ErrNoLiveSessionSandbox), errors.Is(err, sandbox.ErrSandboxPaused):
			c.Error(apperrors.NewConflictError("workspace is not running"))
		default:
			c.Error(apperrors.NewBadRequestError(err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
