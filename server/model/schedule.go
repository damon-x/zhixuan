package model

import "time"

type Schedule struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	Name         string    `gorm:"not null" json:"name"`
	Type         string    `gorm:"not null" json:"type"`
	ScheduleMode string    `gorm:"not null;default:cron" json:"schedule_mode"` // cron 或 once
	Cron         string    `gorm:"not null" json:"cron"`
	Params       string    `gorm:"type:text" json:"params"`
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	QQNotify     bool      `gorm:"default:false" json:"qq_notify"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
