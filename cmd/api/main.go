package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"toggly.com/m/cmd/api/internal/config"
	"toggly.com/m/cmd/api/internal/database"
	"toggly.com/m/cmd/api/internal/seed"
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
	fmt.Printf("CFG %w", cfg)
	db := database.Connect(cfg.DB.ConnectionString())

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	seed.Packets(db, log)

	server.Init(db, cfg)
}
