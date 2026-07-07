package model

import "time"

// Skill 用户自定义提示词，摘要注入上下文，详情经 load_skill tool 懒加载。
type Skill struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Summary   string    `gorm:"type:text" json:"summary"`
	Detail    string    `gorm:"type:text" json:"detail"`
	Enabled   bool      `gorm:"not null;default:false" json:"enabled"`
	Sort      int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
