package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/context"
	"zhixuan/server/database"
	"zhixuan/server/gateway"
	"zhixuan/server/memory"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
)

// ensureMainSession ensures the user has a main session, creating one if needed.
// Used by GetSessions to guarantee main session exists.
func ensureMainSession(userID uint) (*model.Session, error) {
	var session model.Session
	err := database.DB.Where("user_id = ? AND is_main = ?", userID, true).First(&session).Error
	if err == nil {
		return &session, nil
	}

	// Create main session
	session = model.Session{
		UserID:    userID,
		SessionID: model.GenerateSessionID(),
		Title:     "知玄",
		IsMain:    true,
	}
	if err := database.DB.Create(&session).Error; err != nil {
		// Concurrent creation race: re-query
		if err2 := database.DB.Where("user_id = ? AND is_main = ?", userID, true).First(&session).Error; err2 != nil {
			return nil, err
		}
		return &session, nil
	}
	return &session, nil
}

func SendMessage(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		SessionID      string   `json:"session_id" binding:"required"`
		Content        string   `json:"content" binding:"required"`
		WebSearch      bool     `json:"web_search"`
		KnowledgeBases []string `json:"knowledge_bases"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	log.Printf("[chat/send] user=%d session=%s content=%q web_search=%v knowledge_bases=%v",
		user.ID, req.SessionID, req.Content, req.WebSearch, req.KnowledgeBases)

	resultChan := make(chan *gateway.ChatResponse, 1)
	gateway.Get().Chat(&gateway.ChatRequest{
		UserID:         user.ID,
		SessionID:      req.SessionID,
		Content:        req.Content,
		WebSearch:      req.WebSearch,
		KnowledgeBases: req.KnowledgeBases,
		Source:         gateway.SourceWeb,
		ResultChan:     resultChan,
	})

	resp := <-resultChan
	if resp.Error != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": resp.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":         resp.MessageID,
			"role":       "assistant",
			"content":    resp.Content,
			"created_at": resp.CreatedAt,
		},
	})
}

func StopSendMessage(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	gateway.Get().Stop(user.ID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已停止"})
}

func GetSessions(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	// Ensure main session exists
	if _, err := ensureMainSession(user.ID); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "初始化主会话失败"})
		return
	}

	var sessions []model.Session
	database.DB.Where("user_id = ?", user.ID).
		Order("is_main desc, updated_at desc").
		Find(&sessions)

	type SessionInfo struct {
		SessionID  string    `json:"session_id"`
		Title      string    `json:"title"`
		IsMain     bool      `json:"is_main"`
		UpdatedAt  time.Time `json:"updated_at"`
		TopicSince uint      `json:"topic_since"`
	}

	var result []SessionInfo
	for _, s := range sessions {
		result = append(result, SessionInfo{
			SessionID:  s.SessionID,
			Title:      s.Title,
			IsMain:     s.IsMain,
			UpdatedAt:  s.UpdatedAt,
			TopicSince: s.TopicSince,
		})
	}

	if result == nil {
		result = []SessionInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func GetSessionMessages(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	sessionID := c.Param("id")

	// 分页游标：before 为毫秒时间戳，仅加载严格更早的消息；缺省取最新一页
	var beforeTs int64
	if v := c.Query("before"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			beforeTs = n
		}
	}
	rounds := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			rounds = n
		}
	}

	items, hasMore := context.LoadRecentForDisplay(sessionID, rounds, beforeTs)
	for i := range items {
		items[i].UserID = user.ID
		items[i].SessionID = sessionID
	}
	if items == nil {
		items = []context.HistoryItem{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": items,
		"has_more": hasMore,
	})
}

func CreateSession(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	sessionID := model.GenerateSessionID()
	session := model.Session{
		UserID:    user.ID,
		SessionID: sessionID,
		Title:     "",
		IsMain:    false,
	}
	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建会话失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"session_id": sessionID,
		},
	})
}

func DeleteSession(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	sessionID := c.Param("id")

	// Find session to check is_main
	var session model.Session
	if err := database.DB.Where("user_id = ? AND session_id = ?", user.ID, sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "会话不存在"})
		return
	}

	if session.IsMain {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "主会话不可删除"})
		return
	}

	if err := database.DB.Delete(&session).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "删除失败"})
		return
	}

	// 清理会话相关的历史文件和上下文缓存
	context.DeleteSessionHistory(sessionID)
	context.Invalidate(sessionID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已删除"})
}

func StartTopic(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	sessionID := c.Param("id")
	var session model.Session
	if err := database.DB.Where("user_id = ? AND session_id = ?", user.ID, sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "会话不存在"})
		return
	}

	// topic_since 改为当前毫秒时间戳，新 topic 的消息 CreatedAt 必然晚于此时刻
	topicSince := uint(time.Now().UnixMilli())
	database.DB.Model(&session).Update("topic_since", topicSince)
	context.Invalidate(sessionID)
	memory.ResetRecallWindow(sessionID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已开始新话题", "data": gin.H{"topic_since": topicSince}})
}

func UploadChatImage(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "请选择图片"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "仅支持 jpg/jpeg/png/gif 格式"})
		return
	}

	userDir := filepath.Join(config.UploadDir(), fmt.Sprintf("%d", user.ID))
	os.MkdirAll(userDir, 0755)

	filename := fmt.Sprintf("%d%s", time.Now().UnixMilli(), ext)
	savePath := filepath.Join(userDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "保存失败"})
		return
	}

	imagePath := fmt.Sprintf("upload@%d/%s", user.ID, filename)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"image_path": imagePath}})
}
