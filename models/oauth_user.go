package models

import "time"

type OAuthUser struct {
	ID          uint   `gorm:"primaryKey"`
	Email       string `gorm:"uniqueIndex"`
	TwoFASecret string
	Username    string       `json:"username"`
	Avatar      string       `json:"avatar"`
	Provider    string       `json:"provider"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
