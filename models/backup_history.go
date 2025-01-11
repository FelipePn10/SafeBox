package models

import "time"

// Backup represents a backup entry in the database
type Backup struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"not null"`
	AppName  string `gorm:"not null"`
	FilePath string `gorm:"not null"`
}

// BackupHistory represents the history of backups in the database
type BackupHistory struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"not null"`
	AppName    string    `gorm:"not null"`
	BackupDate time.Time `gorm:"not null"`
	BackupMode string    `gorm:"not null"`
	FilePath   string    `gorm:"not null"`
}
