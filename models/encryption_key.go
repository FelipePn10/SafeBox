package models

import (
	"time"
)

type EncryptionKey struct {
	ID        uint      `gorm:"primaryKey"`
	FilePath  string    `gorm:"uniqueIndex;not null"` // Caminho do arquivo associado à chave
	Key       string    `gorm:"not null"`             // Chave de criptografia
	CreatedAt time.Time `gorm:"autoCreateTime"`       // Data de criação
}
