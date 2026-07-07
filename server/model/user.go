package model

import "time"

type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Password  string    `gorm:"not null" json:"-"`

	QQBotAppID     string `gorm:"column:qq_bot_app_id" json:"qq_bot_app_id"`
	QQBotAppSecret string `gorm:"column:qq_bot_app_secret" json:"-"`
	QQBotOpenID        string `gorm:"column:qq_bot_open_id" json:"qq_bot_open_id"`
	QQBotChatEnabled   bool   `gorm:"column:qq_bot_chat_enabled;default:false" json:"qq_bot_chat_enabled"`

	WeChatBound       bool   `gorm:"column:wechat_bound;default:false" json:"wechat_bound"`
	WeChatChatEnabled bool   `gorm:"column:wechat_chat_enabled;default:false" json:"wechat_chat_enabled"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
