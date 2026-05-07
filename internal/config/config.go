package config

import (
	"time"

	"bakery/internal/pkg/helpers"
)

type TelegramConfig struct {
	OrderBotToken string
}

type IikoConfig struct {
	Host     string
	Port     int
	Login    string
	Password string
}

type Config struct {
	Telegram    TelegramConfig
	Iiko        IikoConfig
	DatabaseURL string
	Sync        SyncConfig
}

type SyncConfig struct {
	Interval time.Duration
	DateFrom string
	DateTo   string
}

func New() *Config {
	return &Config{
		Telegram: TelegramConfig{
			OrderBotToken: helpers.Env("ORDER_BOT_TOKEN", helpers.Env("BOT_TOKEN", "")),
		},
		Iiko: IikoConfig{
			Host:     helpers.Env("IIKO_HOST", ""),
			Port:     helpers.EnvInt("IIKO_PORT", 443),
			Login:    helpers.Env("IIKO_LOGIN", ""),
			Password: helpers.Env("IIKO_PASSWORD", ""),
		},
		DatabaseURL: helpers.Env("DATABASE_URL", helpers.Env("DB_DSN", "postgres://postgres:postgres@localhost:5432/bakery?sslmode=disable")),
		Sync: SyncConfig{
			Interval: helpers.EnvDuration("SYNC_INTERVAL", 6*time.Hour),
			DateFrom: helpers.Env("SYNC_DATE_FROM", time.Now().Format("2006-01-02")),
			DateTo:   helpers.Env("SYNC_DATE_TO", time.Now().Format("2006-01-02")),
		},
	}
}
