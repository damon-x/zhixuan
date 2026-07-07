package model

import "time"

type ScheduleLog struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	ScheduleID uint      `gorm:"index;not null" json:"schedule_id"`
	Result     string    `gorm:"type:text" json:"result"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
}
