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

	db.AutoMigrate(
		&domain.User{},
		&domain.Project{},
		&domain.Environment{},
		&domain.Flag{},
		&domain.FlagRule{},
		&domain.APIKey{},
		// Analytics (rollups + activity feed)
		&domain.AnalyticsAPIUsageDaily{},
		&domain.AnalyticsEnvEvaluationsDaily{},
		&domain.AnalyticsFlagEvaluationsDaily{},
		&domain.AnalyticsActivityEvent{},
	)

	return db
}
