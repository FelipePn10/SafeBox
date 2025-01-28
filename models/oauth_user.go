package models

import (
	"time"

	"gorm.io/gorm"
)

type OAuthUser struct {
	gorm.Model
	Password     string
	Email        string `gorm:"unique;not null"`
	Username     string `gorm:"not null"`
	OAuthID      string `gorm:"column:oauth_id"`
	Avatar       string
	Provider     string
	Permissions  []PermissionModel `gorm:"many2many:user_permissions;"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StorageUsed  int64
	StorageLimit int64
	Plan         string
	Backups      []Backup `gorm:"foreignKey:UserID"`
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
}
