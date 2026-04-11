package config

import (
	"fmt"
	"os"
)

type Postgres struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

func (p Postgres) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB)
}

type Iiko struct {
	Host     string
	Port     string
	Login    string
	Password string
}

type Config struct {
	Postgres Postgres
	Iiko     Iiko
}

func Load() (*Config, error) {
	cfg := &Config{
		Postgres: Postgres{
			Host:     env("POSTGRES_HOST", "localhost"),
			Port:     env("POSTGRES_PORT", "5432"),
			User:     env("POSTGRES_USER", "bakery"),
			Password: env("POSTGRES_PASSWORD", ""),
			DB:       env("POSTGRES_DB", "bakery"),
		},
		Iiko: Iiko{
			Host:     env("IIKO_HOST", ""),
			Port:     env("IIKO_PORT", "443"),
			Login:    env("IIKO_LOGIN", ""),
			Password: env("IIKO_PASSWORD", ""),
		},
	}

	if cfg.Iiko.Host == "" {
		return nil, fmt.Errorf("config: IIKO_HOST is required")
	}
	if cfg.Iiko.Login == "" {
		return nil, fmt.Errorf("config: IIKO_LOGIN is required")
	}
	if cfg.Iiko.Password == "" {
		return nil, fmt.Errorf("config: IIKO_PASSWORD is required")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
