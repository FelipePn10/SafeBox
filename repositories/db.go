package repositories

import (
	"SafeBox/models"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DBConection *gorm.DB

type DBConfig struct {
	DB_HOST     string
	DB_USER     string
	DB_PASSWORD string
	DB_NAME     string
	DB_PORT     string
}

func InitDB(cfg DBConfig) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DB_HOST, cfg.DB_USER, cfg.DB_PASSWORD, cfg.DB_NAME, cfg.DB_PORT)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatal("Error connecting to database ", err)
	}
	DBConection = db

	db.AutoMigrate(&models.User{}, &models.OAuthUser{}, &models.PermissionModel{})

	logrus.Info("Database migration completed successfully")
}
