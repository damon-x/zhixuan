package handler

import (
	"net/http"
	"strconv"

	"zhixuan/server/database"
	"zhixuan/server/model"

	"github.com/gin-gonic/gin"
)

func CreateSkill(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required"`
		Summary string `json:"summary"`
		Detail  string `json:"detail"`
		Enabled bool   `json:"enabled"`
		Sort    int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	skill := model.Skill{
		UserID:  user.ID,
		Name:    req.Name,
		Summary: req.Summary,
		Detail:  req.Detail,
		Enabled: req.Enabled,
		Sort:    req.Sort,
	}
	if err := database.DB.Create(&skill).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": skill})
}

func GetSkills(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var skills []model.Skill
	if err := database.DB.Where("user_id = ?", user.ID).Order("sort asc, updated_at desc").Find(&skills).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "查询失败"})
		return
	}
	if skills == nil {
		skills = []model.Skill{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": skills})
}

func UpdateSkill(c *gin.Context) {
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
		Name    string `json:"name" binding:"required"`
		Summary string `json:"summary"`
		Detail  string `json:"detail"`
		Enabled *bool  `json:"enabled"`
		Sort    *int   `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	var skill model.Skill
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&skill).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "skill 不存在"})
		return
	}

	skill.Name = req.Name
	skill.Summary = req.Summary
	skill.Detail = req.Detail
	if req.Enabled != nil {
		skill.Enabled = *req.Enabled
	}
	if req.Sort != nil {
		skill.Sort = *req.Sort
	}
	if err := database.DB.Save(&skill).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": skill})
}

func ToggleSkill(c *gin.Context) {
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
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	result := database.DB.Model(&model.Skill{}).Where("id = ? AND user_id = ?", id, user.ID).Update("enabled", req.Enabled)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "skill 不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func DeleteSkill(c *gin.Context) {
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

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&model.Skill{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "skill 不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
