package repositories

import (
	"SafeBox/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DBConection *gorm.DB

func InitDB() {
	dsn := "host=localhost user=usuario password=1234 dbname=safeboxdb port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatal("Error connecting to database ", err)
	}
	DBConection = db
	db.AutoMigrate(&models.User{})
}
