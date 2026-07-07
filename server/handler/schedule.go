package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"zhixuan/server/database"
	"zhixuan/server/model"
	"zhixuan/server/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func validateScheduleTime(mode, cronStr string) error {
	if mode == "once" {
		t, err := time.ParseInLocation("2006-01-02 15:04", cronStr, time.Local)
		if err != nil {
			return fmt.Errorf("时间格式不合法，请使用 YYYY-MM-DD HH:mm 格式")
		}
		if time.Until(t) <= 0 {
			return fmt.Errorf("目标时间已过")
		}
		return nil
	}
	if _, err := cron.ParseStandard(cronStr); err != nil {
		return fmt.Errorf("cron 表达式不合法")
	}
	return nil
}

func CreateSchedule(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Name         string `json:"name" binding:"required"`
		Type         string `json:"type" binding:"required"`
		ScheduleMode string `json:"schedule_mode"`
		Cron         string `json:"cron" binding:"required"`
		Params       string `json:"params"`
		QQNotify     bool   `json:"qq_notify"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	if req.ScheduleMode == "" {
		req.ScheduleMode = "cron"
	}

	if err := validateScheduleTime(req.ScheduleMode, req.Cron); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	sched := model.Schedule{
		UserID:       user.ID,
		Name:         req.Name,
		Type:         req.Type,
		ScheduleMode: req.ScheduleMode,
		Cron:         req.Cron,
		Params:       req.Params,
		Enabled:      true,
		QQNotify:     req.QQNotify,
	}
	if err := database.DB.Create(&sched).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建失败"})
		return
	}

	scheduler.AddJob(&sched)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": sched})
}

func GetSchedules(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var schedules []model.Schedule
	if err := database.DB.Where("user_id = ?", user.ID).Order("created_at desc").Find(&schedules).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": schedules})
}

func UpdateSchedule(c *gin.Context) {
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
		Name         string `json:"name" binding:"required"`
		Type         string `json:"type" binding:"required"`
		ScheduleMode string `json:"schedule_mode"`
		Cron         string `json:"cron" binding:"required"`
		Params       string `json:"params"`
		Enabled      *bool  `json:"enabled"`
		QQNotify     *bool  `json:"qq_notify"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	if req.ScheduleMode == "" {
		req.ScheduleMode = "cron"
	}

	if err := validateScheduleTime(req.ScheduleMode, req.Cron); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	var sched model.Schedule
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&sched).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "任务不存在"})
		return
	}

	sched.Name = req.Name
	sched.Type = req.Type
	sched.ScheduleMode = req.ScheduleMode
	sched.Cron = req.Cron
	sched.Params = req.Params
	if req.Enabled != nil {
		sched.Enabled = *req.Enabled
	}
	if req.QQNotify != nil {
		sched.QQNotify = *req.QQNotify
	}

	if err := database.DB.Save(&sched).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}

	scheduler.RemoveJob(sched.ID)
	if sched.Enabled {
		scheduler.AddJob(&sched)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": sched})
}

func DeleteSchedule(c *gin.Context) {
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

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&model.Schedule{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "任务不存在"})
		return
	}

	scheduler.RemoveJob(uint(id))

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func GetScheduleTypes(c *gin.Context) {
	types := []gin.H{
		{"value": "agent", "label": "Agent 任务"},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": types})
}

func GetScheduleLogs(c *gin.Context) {
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

	var sched model.Schedule
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&sched).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "任务不存在"})
		return
	}

	var logs []model.ScheduleLog
	database.DB.Where("schedule_id = ?", id).Order("created_at desc").Limit(50).Find(&logs)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": logs})
}
