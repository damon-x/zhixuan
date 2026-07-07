package handler

import (
	"net/http"

	"zhixuan/server/database"
	"zhixuan/server/gateway"
	"zhixuan/server/model"
	"zhixuan/server/qqbot"

	"github.com/gin-gonic/gin"
)

func StartQQBotBind(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		AppID     string `json:"app_id" binding:"required"`
		AppSecret string `json:"app_secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	// Save credentials to user record
	database.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"qq_bot_app_id":     req.AppID,
		"qq_bot_app_secret": req.AppSecret,
		"qq_bot_open_id":    "",
	})

	code, err := qqbot.StartBinding(user.ID, req.AppID, req.AppSecret)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "启动绑定失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"code": code}})
}

func CheckQQBotBind(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	done, openID := qqbot.CheckBinding(user.ID)
	if done && openID != "" {
		// Persist openid to user record
		database.DB.Model(&model.User{}).Where("id = ?", user.ID).Update("qq_bot_open_id", openID)
		qqbot.CancelBinding(user.ID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"bound":  done && openID != "",
		"openid": openID,
	}})
}

func GetQQBotStatus(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"bound":  user.QQBotOpenID != "",
		"app_id": user.QQBotAppID,
	}})
}

func ToggleQQBotChat(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	database.DB.Model(&model.User{}).Where("id = ?", user.ID).Update("qq_bot_chat_enabled", req.Enabled)

	if req.Enabled {
		if err := gateway.Get().StartQQChat(user.ID); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "启动 QQ 对话失败: " + err.Error()})
			return
		}
	} else {
		gateway.Get().StopQQChat(user.ID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已更新"})
}

func GetQQBotChatStatus(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"enabled": user.QQBotChatEnabled,
		"running": gateway.Get().IsQQChatRunning(user.ID),
	}})
}
