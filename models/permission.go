package models

import "gorm.io/gorm"

type Permission string

const (
	READ             Permission = "read"
	WRITE            Permission = "write"
	DELETE           Permission = "delete"
	ADMIN            Permission = "admin"
	PermissionBackup            = "backup"
)

type PermissionModel struct {
	gorm.Model
	Name string `json:"name" gorm:"unique"`
}
