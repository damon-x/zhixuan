package handler

import (
	"net/http"
	"strconv"

	"zhixuan/server/database"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
)

func CreateNote(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
		PlanID  *uint  `json:"plan_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	note := model.Note{
		UserID:  user.ID,
		Title:   req.Title,
		Content: req.Content,
		PlanID:  req.PlanID,
	}
	if err := database.DB.Create(&note).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": note})
}

func GetNotes(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var notes []model.Note
	keyword := c.Query("keyword")
	planID := c.Query("plan_id")
	q := database.DB.Where("user_id = ?", user.ID).
		Select("id, title, updated_at")
	if keyword != "" {
		q = q.Where("title LIKE ?", "%"+keyword+"%")
	}
	if planID != "" {
		q = q.Where("plan_id = ?", planID)
	}
	if err := q.Order("updated_at desc").Find(&notes).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": notes})
}

func GetNote(c *gin.Context) {
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

	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&note).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "笔记不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": note})
}

func UpdateNote(c *gin.Context) {
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&note).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "笔记不存在"})
		return
	}

	note.Title = req.Title
	note.Content = req.Content
	if err := database.DB.Save(&note).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": note})
}

func DeleteNote(c *gin.Context) {
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

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&model.Note{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "笔记不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
