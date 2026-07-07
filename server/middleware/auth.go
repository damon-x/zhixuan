package middleware

import (
	"net/http"

	"zhixuan/server/handler"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
			c.Abort()
			return
		}
		_, ok := handler.GetUserIDByToken(token)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "登录已过期"})
			c.Abort()
			return
		}
		c.Next()
	}
}
