package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 24 * time.Hour

func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "系统错误"})
		return
	}

	user := model.User{Username: req.Username, Password: string(hash)}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "用户名已存在"})
		return
	}

	// Create main session for new user
	session := model.Session{
		UserID:    user.ID,
		SessionID: model.GenerateSessionID(),
		Title:     "知玄",
		IsMain:    true,
	}
	database.DB.Create(&session)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "注册成功"})
}

func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	var user model.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "用户名或密码错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "用户名或密码错误"})
		return
	}

	tokenStr := generateToken()
	database.DB.Create(&model.Token{
		Token:     tokenStr,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(tokenTTL),
	})

	c.SetCookie("token", tokenStr, config.TokenMaxAge, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "登录成功",
		"data": gin.H{"username": user.Username},
	})
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func Logout(c *gin.Context) {
	token, err := c.Cookie("token")
	if err == nil && token != "" {
		database.DB.Where("token = ?", token).Delete(&model.Token{})
	}
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已退出登录"})
}

func GetUserIDByToken(tokenStr string) (uint, bool) {
	var t model.Token
	if err := database.DB.Where("token = ? AND expires_at > ?", tokenStr, time.Now()).First(&t).Error; err != nil {
		return 0, false
	}
	return t.UserID, true
}

func GetCurrentUser(c *gin.Context) (*model.User, bool) {
	token, err := c.Cookie("token")
	if err != nil {
		return nil, false
	}
	userID, ok := GetUserIDByToken(token)
	if !ok {
		return nil, false
	}
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, false
	}
	return &user, true
}
