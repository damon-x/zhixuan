package model

import (
	"crypto/rand"
	"fmt"
	"time"
)

type Session struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	SessionID string    `gorm:"uniqueIndex:idx_user_session;not null" json:"session_id"`
	Title     string    `gorm:"not null;default:''" json:"title"`
	IsMain     bool      `gorm:"not null;default:false" json:"is_main"`
	TopicSince uint      `gorm:"not null;default:0" json:"topic_since"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func GenerateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
