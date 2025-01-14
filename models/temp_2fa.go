package models

import "time"

type TempTwoFASecret struct {
	ID        uint   `gorm:"primaryKey"`
	UserEmail string `gorm:"uniqueIndex"`
	Secret    string
	CreatedAt time.Time
}
