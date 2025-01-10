package models

import "time"

type OAuthUser struct {
	ID          string       `json:"id"`
	Email       string       `json:"email"`
	Username    string       `json:"username"`
	Avatar      string       `json:"avatar"`
	Provider    string       `json:"provider"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
