package model

import "time"

type Todo struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	UserID    uint       `gorm:"index;not null" json:"user_id"`
	Title     string     `gorm:"not null" json:"title"`
	Content   string     `gorm:"type:text" json:"content"`
	PlanID    *uint      `gorm:"index" json:"plan_id"`
	Priority  int        `gorm:"not null;default:0" json:"priority"`
	Deadline  *time.Time `json:"deadline"`
	Done      bool       `gorm:"default:false" json:"done"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
