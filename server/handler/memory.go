package handler

import (
	stdctx "context"
	"net/http"
	"strconv"

	"zhixuan/server/memory"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
)

// GetMemories 返回当前用户的全部记忆，按时间倒序。
func GetMemories(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}
	mems, err := memory.Get().List(user.ID, 200)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "查询失败"})
		return
	}
	if mems == nil {
		mems = []model.Memory{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": mems})
}

// DeleteMemory 删除一条记忆（含向量索引）。
func DeleteMemory(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}
	if err := memory.Get().Delete(stdctx.Background(), user.ID, uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
