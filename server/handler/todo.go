package handler

import (
	"net/http"
	"strconv"
	"time"

	"zhixuan/server/database"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
)

func CreateTodo(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content"`
		Priority int    `json:"priority"`
		Deadline string `json:"deadline"`
		PlanID   *uint  `json:"plan_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	todo := model.Todo{
		UserID:   user.ID,
		Title:    req.Title,
		Content:  req.Content,
		Priority: req.Priority,
		PlanID:   req.PlanID,
	}
	if req.Deadline != "" {
		t, err := time.Parse("2006-01-02", req.Deadline)
		if err == nil {
			todo.Deadline = &t
		}
	}

	if err := database.DB.Create(&todo).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": todo})
}

func GetTodos(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var todos []model.Todo
	planID := c.Query("plan_id")
	q := database.DB.Where("user_id = ?", user.ID)
	if planID != "" {
		q = q.Where("plan_id = ?", planID)
	}
	if err := q.Order("done asc, priority desc, created_at asc").
		Find(&todos).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": todos})
}

func UpdateTodo(c *gin.Context) {
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

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content"`
		Priority int    `json:"priority"`
		Deadline string `json:"deadline"`
		Done     *bool  `json:"done"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	var todo model.Todo
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&todo).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "待办不存在"})
		return
	}

	todo.Title = req.Title
	todo.Content = req.Content
	todo.Priority = req.Priority
	if req.Done != nil {
		todo.Done = *req.Done
	}
	if req.Deadline != "" {
		t, err := time.Parse("2006-01-02", req.Deadline)
		if err == nil {
			todo.Deadline = &t
		}
	} else {
		todo.Deadline = nil
	}

	if err := database.DB.Save(&todo).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": todo})
}

func DeleteTodo(c *gin.Context) {
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

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&model.Todo{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "待办不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
