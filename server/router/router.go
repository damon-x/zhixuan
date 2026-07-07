package router

import (
	"mime"
	"path/filepath"
	"strings"

	"zhixuan/server/handler"
	"zhixuan/server/middleware"
	"zhixuan/server/static"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()

	// public routes
	r.POST("/api/register", handler.Register)
	r.POST("/api/login", handler.Login)
	r.POST("/api/logout", handler.Logout)

	// protected routes
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		api.GET("/me", func(c *gin.Context) {
			user, ok := handler.GetCurrentUser(c)
			if !ok {
				c.JSON(401, gin.H{"code": 1, "msg": "未登录"})
				return
			}
			c.JSON(200, gin.H{"code": 0, "msg": "success", "data": user})
		})

		api.POST("/notes", handler.CreateNote)
		api.GET("/notes", handler.GetNotes)
		api.GET("/notes/:id", handler.GetNote)
		api.PUT("/notes/:id", handler.UpdateNote)
		api.DELETE("/notes/:id", handler.DeleteNote)

		api.POST("/todos", handler.CreateTodo)
		api.GET("/todos", handler.GetTodos)
		api.PUT("/todos/:id", handler.UpdateTodo)
		api.DELETE("/todos/:id", handler.DeleteTodo)

		api.POST("/plans", handler.CreatePlan)
		api.GET("/plans", handler.GetPlans)
		api.GET("/plans/:id", handler.GetPlan)
		api.PUT("/plans/:id", handler.UpdatePlan)
		api.DELETE("/plans/:id", handler.DeletePlan)

		api.POST("/chat/send", handler.SendMessage)
		api.POST("/chat/upload", handler.UploadChatImage)
		api.POST("/chat/stop", handler.StopSendMessage)
		api.GET("/chat/sessions", handler.GetSessions)
		api.POST("/chat/sessions", handler.CreateSession)
		api.GET("/chat/sessions/:id", handler.GetSessionMessages)
		api.DELETE("/chat/sessions/:id", handler.DeleteSession)
		api.PUT("/chat/sessions/:id/topic", handler.StartTopic)

		api.POST("/ai/task", handler.ExecuteAiTask)

		api.POST("/knowledge-bases", handler.CreateKB)
		api.GET("/knowledge-bases", handler.ListKB)
		api.PUT("/knowledge-bases/:name", handler.UpdateKB)
		api.DELETE("/knowledge-bases/:name", handler.DeleteKB)
		api.GET("/knowledge-bases/:name/docs", handler.ListDocs)
		api.POST("/knowledge-bases/:name/docs", handler.UploadDoc)
		api.GET("/knowledge-bases/:name/docs/:doc", handler.PreviewDoc)
		api.DELETE("/knowledge-bases/:name/docs/:doc", handler.DeleteDoc)

		api.GET("/resource", handler.GetResource)

		api.POST("/qqbot/bind", handler.StartQQBotBind)
		api.GET("/qqbot/bind/check", handler.CheckQQBotBind)
		api.GET("/qqbot/status", handler.GetQQBotStatus)
		api.POST("/qqbot/chat/toggle", handler.ToggleQQBotChat)
		api.GET("/qqbot/chat/status", handler.GetQQBotChatStatus)

		api.POST("/wechat/qrcode", handler.GetWeChatQRCode)
		api.GET("/wechat/bind/status", handler.CheckWeChatBind)
		api.GET("/wechat/status", handler.GetWeChatStatus)
		api.POST("/wechat/chat/toggle", handler.ToggleWeChatChat)
		api.GET("/wechat/chat/status", handler.GetWeChatChatStatus)

		api.POST("/schedules", handler.CreateSchedule)
		api.GET("/schedules", handler.GetSchedules)
		api.PUT("/schedules/:id", handler.UpdateSchedule)
		api.DELETE("/schedules/:id", handler.DeleteSchedule)
		api.GET("/schedules/types", handler.GetScheduleTypes)
		api.GET("/schedules/:id/logs", handler.GetScheduleLogs)

		api.GET("/memories", handler.GetMemories)
		api.DELETE("/memories/:id", handler.DeleteMemory)

		api.POST("/skills", handler.CreateSkill)
		api.GET("/skills", handler.GetSkills)
		api.PUT("/skills/:id", handler.UpdateSkill)
		api.PUT("/skills/:id/toggle", handler.ToggleSkill)
		api.DELETE("/skills/:id", handler.DeleteSkill)
	}

	// Serve embedded frontend (SPA). Must come after all API routes.
	r.NoRoute(serveStatic)

	return r
}

// serveStatic serves the embedded frontend build.
//   - /api/* misses → JSON 404
//   - exact file exists in embed → returned with content-type
//   - otherwise → index.html (SPA hash-router fallback)
//   - index.html missing (dev build with empty dist) → JSON 404
func serveStatic(c *gin.Context) {
	p := c.Request.URL.Path
	if strings.HasPrefix(p, "/api/") {
		c.JSON(404, gin.H{"code": 1, "msg": "not found"})
		return
	}

	// Map request path to embed path ("dist/..."). Root → index.html.
	rel := strings.TrimPrefix(p, "/")
	embedPath := "dist/" + rel
	if rel == "" {
		embedPath = "dist/index.html"
	}

	if data, err := static.Dist.ReadFile(embedPath); err == nil {
		c.Data(200, contentType(embedPath), data)
		return
	}

	// SPA fallback.
	if data, err := static.Dist.ReadFile("dist/index.html"); err == nil {
		c.Data(200, "text/html; charset=utf-8", data)
		return
	}

	// Dev mode: dist is empty (only .gitkeep).
	c.JSON(404, gin.H{"code": 1, "msg": "前端未构建"})
}

// contentType resolves the Content-Type for an embed path by extension.
func contentType(path string) string {
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
