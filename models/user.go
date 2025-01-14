package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
)

type User struct {
	gorm.Model
	Username     string       `gorm:"unique;not null"`
	Password     string       `gorm:"not null"`
	Email        string       `gorm:"unique;not null"`
	StorageUsed  int64        `gorm:"default:0"`
	StorageLimit int64        `gorm:"default:1073741824"`
	Plan         string       `gorm:"default:'free'"`
	Permissions  []Permission `gorm:"type:text[]"`
	TwoFASecret  string       `gorm:"default:''"`
	Backups      []Backup     `gorm:"foreignKey:UserID"`
}

func (user *User) BeforeSave(tx *gorm.DB) (err error) {
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.Password = string(hashedPassword)
	}
	return nil
}
