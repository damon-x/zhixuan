package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/gateway"
	"zhixuan/server/model"
	"zhixuan/server/wechat"

	"github.com/gin-gonic/gin"
)

// wechatBindState tracks in-progress QR binding sessions
type wechatBindState struct {
	QRCode string
	UserID uint
	Done   bool
}

var (
	wxBindMu  sync.Mutex
	wxBindMap = map[uint]*wechatBindState{} // key: userID
)

// GetWeChatQRCode handles POST /api/wechat/qrcode
func GetWeChatQRCode(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	qrcode, qrImage, err := wechat.FetchQRCode()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "获取二维码失败: " + err.Error()})
		return
	}

	statePath := wechat.StatePath(config.DataDir, user.ID)

	// Store binding state
	wxBindMu.Lock()
	wxBindMap[user.ID] = &wechatBindState{
		QRCode: qrcode,
		UserID: user.ID,
		Done:   false,
	}
	wxBindMu.Unlock()

	// Start background goroutine to poll QR status
	go pollWeChatQRStatus(user.ID, qrcode, statePath)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"qrcode":   qrcode,
			"qr_image": qrImage,
		},
	})
}

func pollWeChatQRStatus(userID uint, qrcode string, statePath string) {
	deadline := time.Now().Add(8 * time.Minute)

	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		status, botToken, accountID, wxUserID, baseURL, err := wechat.PollQRStatus(qrcode)
		if err != nil {
			continue
		}

		switch status {
		case "confirmed":
			if botToken == "" || accountID == "" {
				log.Printf("[wechat] 用户 %d 扫码确认但缺少关键字段", userID)
				continue
			}
			// Save state to file
			if err := wechat.SaveNewState(statePath, botToken, accountID, wxUserID, baseURL); err != nil {
				log.Printf("[wechat] 保存 state 失败: %v", err)
				continue
			}
			// Mark user as bound
			database.DB.Model(&model.User{}).Where("id = ?", userID).Update("wechat_bound", true)

			wxBindMu.Lock()
			if s, ok := wxBindMap[userID]; ok {
				s.Done = true
			}
			wxBindMu.Unlock()

			log.Printf("[wechat] 用户 %d 绑定成功", userID)
			return
		case "expired", "verify_code_blocked":
			wxBindMu.Lock()
			delete(wxBindMap, userID)
			wxBindMu.Unlock()
			return
		}
	}

	// Timeout
	wxBindMu.Lock()
	delete(wxBindMap, userID)
	wxBindMu.Unlock()
}

// CheckWeChatBind handles GET /api/wechat/bind/status
func CheckWeChatBind(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	wxBindMu.Lock()
	s, exists := wxBindMap[user.ID]
	bound := exists && s.Done
	wxBindMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"bound": bound,
		},
	})
}

// GetWeChatStatus handles GET /api/wechat/status
func GetWeChatStatus(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"bound": user.WeChatBound,
		},
	})
}

// ToggleWeChatChat handles POST /api/wechat/chat/toggle
func ToggleWeChatChat(c *gin.Context) {
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

	database.DB.Model(&model.User{}).Where("id = ?", user.ID).Update("wechat_chat_enabled", req.Enabled)

	if req.Enabled {
		if err := gateway.Get().StartWeChatChat(user.ID); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "启动微信对话失败: " + err.Error()})
			return
		}
	} else {
		gateway.Get().StopWeChatChat(user.ID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已更新"})
}

// GetWeChatChatStatus handles GET /api/wechat/chat/status
func GetWeChatChatStatus(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"enabled": user.WeChatChatEnabled,
			"running": gateway.Get().IsWeChatChatRunning(user.ID),
		},
	})
}
