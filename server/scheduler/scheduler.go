package scheduler

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"zhixuan/server/database"
	"zhixuan/server/gateway"
	"zhixuan/server/model"
	"zhixuan/server/qqbot"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron   *cron.Cron
	mu     sync.Mutex
	jobs   map[uint]cron.EntryID
	timers map[uint]*time.Timer
}

var instance *Scheduler

func Init() {
	instance = &Scheduler{
		cron:   cron.New(),
		jobs:   make(map[uint]cron.EntryID),
		timers: make(map[uint]*time.Timer),
	}

	var schedules []model.Schedule
	database.DB.Where("enabled = ?", true).Find(&schedules)
	for _, s := range schedules {
		if err := instance.addJob(&s); err != nil {
			log.Printf("[scheduler] 注册任务 %d 失败: %v", s.ID, err)
		}
	}

	instance.cron.Start()
	log.Printf("[scheduler] 已启动，加载了 %d 个任务", len(schedules))
}

func AddJob(sched *model.Schedule) error {
	if instance == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return instance.addJob(sched)
}

func RemoveJob(scheduleID uint) {
	if instance == nil {
		return
	}
	instance.removeJob(scheduleID)
}

func (s *Scheduler) addJob(sched *model.Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !sched.Enabled {
		return nil
	}

	s.removeJobLocked(sched.ID)

	switch sched.ScheduleMode {
	case "once":
		targetTime, err := time.ParseInLocation("2006-01-02 15:04", sched.Cron, time.Local)
		if err != nil {
			return fmt.Errorf("invalid datetime format: %w", err)
		}
		duration := time.Until(targetTime)
		if duration <= 0 {
			return fmt.Errorf("目标时间已过")
		}
		timer := time.AfterFunc(duration, func() {
			executeSchedule(sched)
			// 执行后自动禁用
			database.DB.Model(&model.Schedule{}).Where("id = ?", sched.ID).Update("enabled", false)
		})
		s.timers[sched.ID] = timer
		log.Printf("[scheduler] 注册单次任务 %d (%s): %s", sched.ID, sched.Name, sched.Cron)

	default: // cron
		entryID, err := s.cron.AddFunc(sched.Cron, func() {
			executeSchedule(sched)
		})
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		s.jobs[sched.ID] = entryID
		log.Printf("[scheduler] 注册定时任务 %d (%s): %s", sched.ID, sched.Name, sched.Cron)
	}

	return nil
}

func (s *Scheduler) removeJob(scheduleID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeJobLocked(scheduleID)
}

func (s *Scheduler) removeJobLocked(scheduleID uint) {
	if entryID, ok := s.jobs[scheduleID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
		log.Printf("[scheduler] 移除定时任务 %d", scheduleID)
	}
	if timer, ok := s.timers[scheduleID]; ok {
		timer.Stop()
		delete(s.timers, scheduleID)
		log.Printf("[scheduler] 移除单次任务 %d", scheduleID)
	}
}

func executeSchedule(sched *model.Schedule) {
	log.Printf("[scheduler] 执行任务 %d (%s)", sched.ID, sched.Name)

	var result string
	var errMsg string

	result, errMsg = executeAgentJob(sched)

	scheduleLog := model.ScheduleLog{
		ScheduleID: sched.ID,
		Result:     result,
		Error:      errMsg,
	}
	database.DB.Create(&scheduleLog)

	if sched.QQNotify && result != "" {
		sendQQNotify(sched.UserID, fmt.Sprintf("【定时任务: %s】\n%s", sched.Name, result))
	}
}

func executeAgentJob(sched *model.Schedule) (string, string) {
	var params struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(sched.Params), &params); err != nil {
		return "", fmt.Sprintf("解析参数失败: %v", err)
	}
	if params.Prompt == "" {
		return "", "参数 prompt 为空"
	}

	resultChan := make(chan *gateway.ChatResponse, 1)
	gateway.Get().Chat(&gateway.ChatRequest{
		UserID:     sched.UserID,
		Content:    params.Prompt,
		Source:     gateway.SourceSchedule,
		ResultChan: resultChan,
	})

	resp := <-resultChan
	if resp.Error != nil {
		return "", resp.Error.Error()
	}
	return resp.Content, ""
}

func sendQQNotify(userID uint, message string) {
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		log.Printf("[scheduler] 发送QQ通知失败: 用户 %d 不存在", userID)
		return
	}
	if user.QQBotAppID == "" || user.QQBotAppSecret == "" || user.QQBotOpenID == "" {
		log.Printf("[scheduler] 发送QQ通知失败: 用户 %d 未绑定QQBot", userID)
		return
	}
	if err := qqbot.SendMsg(user.QQBotAppID, user.QQBotAppSecret, user.QQBotOpenID, message); err != nil {
		log.Printf("[scheduler] 发送QQ通知失败: %v", err)
	}
}
