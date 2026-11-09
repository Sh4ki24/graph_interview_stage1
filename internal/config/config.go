package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the application configuration, loaded from config.yaml.
type Config struct {
	Server     Server     `yaml:"server"`
	Database   Database   `yaml:"database"`
	Auth       Auth       `yaml:"auth"`
	Log        Log        `yaml:"log"`
	Pagination Pagination `yaml:"pagination"`
	Seeder     Seeder     `yaml:"seeder"`
}

type Server struct {
	Port        string `yaml:"port"`
	MetricsPort string `yaml:"metrics_port"`
	// Monitoring enables the separate metrics/pprof server.
	Monitoring bool `yaml:"monitoring"`
}

type Database struct {
	URL  string `yaml:"url"`
	Pool Pool   `yaml:"pool"`
}

// Auth holds authentication settings. The JWT secret is intentionally
// NOT stored here — it is read from the JWT_SECRET environment variable.
type Auth struct {
	AdminUsername string `yaml:"admin_username"`
	AdminPassword string `yaml:"admin_password"`
}

// Pool tunes the pgx connection pool. Zero values keep the pgx defaults.
type Pool struct {
	MaxConns        int32         `yaml:"max_conns"`
	MinConns        int32         `yaml:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

// Log configures application logging.
type Log struct {
	// Level is one of debug, info, warn, error.
	Level string `yaml:"level"`
}

// Pagination holds the defaults for GET /tasks.
type Pagination struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
}

// Seeder configures the database seeder (cmd/seed).
type Seeder struct {
	// Count is how many tasks `make seed` inserts.
	Count int `yaml:"count"`
}

// Default returns the built-in defaults.
func Default() Config {
	return Config{
		Server: Server{
			Port:        "8080",
			MetricsPort: "9090",
			Monitoring:  true,
		},
		Database: Database{
			URL:  "postgres://todo:todo@localhost:5433/todo?sslmode=disable",
			Pool: Pool{MaxConns: 10, MinConns: 2, MaxConnLifetime: 30 * time.Minute, MaxConnIdleTime: 5 * time.Minute},
		},
		Auth: Auth{
			AdminUsername: "admin",
			AdminPassword: "admin123",
		},
		Log:        Log{Level: "info"},
		Pagination: Pagination{DefaultLimit: 20, MaxLimit: 100},
		Seeder:     Seeder{Count: 100},
	}
}

// Load reads the yaml config file at path, applying it over the defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = "config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Path returns the config file path, from CONFIG_PATH or the default.
func Path() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return "config.yaml"
}
