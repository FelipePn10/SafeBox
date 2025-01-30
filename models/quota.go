package models

import "fmt"

type StoragePlan string

const (
	Free    StoragePlan = "Free"
	Premium StoragePlan = "Premium"
)

type UserQuota struct {
	ID     uint        `gorm:"primaryKey"`
	UserID uint        `gorm:"uniqueIndex"`
	Limit  int64       // bytes
	Used   int64       // bytes
	Plan   StoragePlan `gorm:"type:varchar(20)"`
}

var ErrStorageLimitExceeded = fmt.Errorf("limite excedido. Atualize seu plano")

func (u *UserQuota) SetDefaults() {
	switch u.Plan {
	case Premium:
		u.Limit = 45 * 1024 * 1024 * 1024 // 45GB
	default:
		u.Limit = 20 * 1024 * 1024 * 1024 // 20GB
	}
}
