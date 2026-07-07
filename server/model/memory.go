package model

import "time"

// 记忆类型
const (
	MemoryTypePreference   = "preference"   // 用户偏好
	MemoryTypeFact         = "fact"         // 重要事实
	MemoryTypeRelationship = "relationship" // 人际关系
	MemoryTypeEvent        = "event"        // 重要事件
	MemoryTypeGoal         = "goal"         // 长期目标
)

// Memory 用户长期记忆，由记忆 agent 异步写入。
type Memory struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Type      string    `gorm:"size:32;not null;default:'fact'" json:"type"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Tags      string    `gorm:"type:text" json:"tags"`       // 逗号分隔的标签
	SessionID string    `gorm:"size:64" json:"session_id"`   // 来源会话
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
