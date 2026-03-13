package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
)

func Connect(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn))

	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&domain.User{})

	return db
}
