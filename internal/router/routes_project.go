package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterProjectRoutes exposes the Codex-style Project/Thread entry point.
// API keys are intentionally excluded: a project root is local-machine
// metadata that must be associated with an authenticated user.
func RegisterProjectRoutes(r *gin.RouterGroup, h *handler.ProjectHandler, g *rbacGuards) {
	if h == nil {
		return
	}
	projects := r.Group("/projects")
	projects.GET("", g.Viewer(), h.List)
	projects.POST("", g.Contributor(), h.Create)
	projects.GET("/:project_id", g.Viewer(), h.Get)
	projects.PUT("/:project_id", g.Contributor(), h.Update)
	projects.DELETE("/:project_id", g.Contributor(), h.Delete)
	projects.GET("/:project_id/folders", g.Viewer(), h.ListFolders)
	projects.POST("/:project_id/folders", g.Contributor(), h.CreateFolder)
	projects.DELETE("/:project_id/folders/:folder_id", g.Contributor(), h.DeleteFolder)
	projects.GET("/:project_id/folders/:folder_id/threads", g.Viewer(), h.ListFolderThreads)
	projects.POST("/:project_id/folders/:folder_id/threads", g.Contributor(), h.CreateFolderThread)
	projects.GET("/:project_id/folders/:folder_id/threads/:thread_id/review", g.Viewer(), h.ReviewThread)
}
