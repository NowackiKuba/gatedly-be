package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	HTTP HTTPConfig `envPrefix:"HTTP_"`
	DB   DBConfig   `envPrefix:"DB_"`
	JWT  JWTConfig  `envPrefix:"JWT_"`
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Addr         string        `env:"ADDR" envDefault:":8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"15s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
}

// DBConfig holds database connection settings.
type DBConfig struct {
	DSN      string `env:"DSN"`
	Host     string `env:"HOST" envDefault:"localhost"`
	User     string `env:"USER" envDefault:"postgres"`
	Password string `env:"PASSWORD" envDefault:""`
	DBName   string `env:"NAME" envDefault:"postgres"`
	Port     string `env:"PORT" envDefault:"5432"`
}

// DSN returns the connection string, building from components if DSN is empty.
func (d *DBConfig) ConnectionString() string {
	if d.DSN != "" {
		return d.DSN
	}
	return "host=" + d.Host + " user=" + d.User + " password=" + d.Password + " dbname=" + d.DBName + " port=" + d.Port + " sslmode=disable"
}

// JWTConfig holds JWT signing and TTL settings.
type JWTConfig struct {
	Secret          string        `env:"SECRET,required"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h"`
}

// MustLoad loads .env from the current directory (ignores error), then parses
// environment into Config. It logs and exits on parse error.
func MustLoad() *Config {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("config: parse env: %v", err)
	}
	return &cfg
}

