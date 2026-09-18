package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config models the environment variables the app needs.
type Config struct {
	AsanaToken           string `env:"ASANA_API_TOKEN,required"`
	DefaultWorkspace     string `env:"ASANA_API_DEFAULT_WORKSPACE,required"`
	LimiterMaxRetries    int    `env:"ASANA_API_LIMITER_MAX_RETRIES" envDefault:"3"`
	MaxRequestsPerMinute int    `env:"ASANA_API_MAXIMUM_REQUESTS_PER_MINUTE" envDefault:"120"`
	PaginationLimit      int    `env:"ASANA_API_PAGINATION_LIMIT" envDefault:"1"`
}

// NewConfig reads .env into the process environment (if present) and parses
// it into a Config.
func NewConfig() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
