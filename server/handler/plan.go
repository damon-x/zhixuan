package handler

import (
	"net/http"
	"strconv"

	"zhixuan/server/database"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
)

func CreatePlan(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	status := req.Status
	if status == "" {
		status = "in_progress"
	}

	plan := model.Plan{
		UserID:  user.ID,
		Title:   req.Title,
		Content: req.Content,
		Status:  status,
	}
	if err := database.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": plan})
}

func GetPlans(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var plans []model.Plan
	if err := database.DB.Where("user_id = ?", user.ID).
		Order("updated_at desc").Find(&plans).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": plans})
}

func GetPlan(c *gin.Context) {
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

	var plan model.Plan
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&plan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "计划不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": plan})
}

func UpdatePlan(c *gin.Context) {
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
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	var plan model.Plan
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&plan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "计划不存在"})
		return
	}

	plan.Title = req.Title
	plan.Content = req.Content
	if req.Status != "" {
		plan.Status = req.Status
	}
	if err := database.DB.Save(&plan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": plan})
}

func DeletePlan(c *gin.Context) {
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

	// 解除关联的笔记、待办（仅置空 plan_id）
	database.DB.Model(&model.Note{}).Where("plan_id = ? AND user_id = ?", id, user.ID).Update("plan_id", nil)
	database.DB.Model(&model.Todo{}).Where("plan_id = ? AND user_id = ?", id, user.ID).Update("plan_id", nil)

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&model.Plan{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "计划不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
