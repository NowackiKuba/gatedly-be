package main

import (
	"github.com/joho/godotenv"
	"toggly.com/m/cmd/api/internal/config"
	"toggly.com/m/cmd/api/internal/database"
	"toggly.com/m/cmd/api/internal/server"
)

func main() {
	// Try loading .env from common paths (run from repo root or cmd/api)
	for _, rel := range []string{".env", ".env", ".env"} {
		if err := godotenv.Load(rel); err == nil {
			break
		}
	}

	cfg := config.MustLoad()
	db := database.Connect(cfg.DB.ConnectionString())

	server.Init(db, cfg)
}
