package model

import "time"

type Chat struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	SessionID string    `gorm:"index;not null" json:"session_id"`
	Role      string    `gorm:"not null" json:"role"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
