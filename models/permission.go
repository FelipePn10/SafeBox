package models

import "gorm.io/gorm"

type Permission string

const (
	READ             Permission = "read"
	WRITE            Permission = "write"
	DELETE           Permission = "delete"
	ADMIN            Permission = "admin"
	PermissionBackup Permission = "backup"
)

type PermissionModel struct {
	gorm.Model
	Name string `gorm:"unique;not null"`
}
