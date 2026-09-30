package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/handler"
)

// RegisterProgrammingRoutes exposes the session-backed programming workspace.
// Session ownership is enforced inside ProgrammingHandler; Viewer keeps the
// route consistent with the existing session surface, while scoped API keys
// need the chat capability to use the Agent/sandbox boundary.
func RegisterProgrammingRoutes(
	r *gin.RouterGroup,
	h *handler.ProgrammingHandler,
	g *rbacGuards,
) {
	// Gin keeps one radix tree per HTTP method. Existing session GET/PUT trees
	// use :id, while the POST tree uses :session_id, so register each method
	// against the matching wildcard name instead of triggering a startup panic.
	getProgramming := g.apiKeyGroup(r.Group("/sessions/:id/programming", g.Viewer()), apiKeyChat(apiKeyFullAccess()))
	getProgramming.GET("/tree", h.ListTree)
	getProgramming.GET("/file", h.ReadFile)
	getProgramming.GET("/preview", h.PreviewCapabilities)
	putProgramming := g.apiKeyGroup(r.Group("/sessions/:id/programming", g.Viewer()), apiKeyChat(apiKeyFullAccess()))
	putProgramming.PUT("/file", h.WriteFile)
	postProgramming := g.apiKeyGroup(r.Group("/sessions/:session_id/programming", g.Viewer()), apiKeyChat(apiKeyFullAccess()))
	postProgramming.POST("/file", h.CreateFile)
	postProgramming.POST("/file/move", h.MoveFile)
	postProgramming.POST("/file/delete", h.DeleteFile)
	postProgramming.POST("/command", h.RunCommand)
	postProgramming.POST("/preview", h.OpenPreview)
	postProgramming.POST("/web-preview", h.DiscoverWebPreview)
}
