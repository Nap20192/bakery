package config

import (
	"fmt"
	"time"

	"bakery/internal/pkg/helpers"
)

type TelegramConfig struct {
	BotEnv       string
	TestBotToken string
	ProdBotToken string
	BotToken     string
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
}

func New() *Config {
	botEnv := helpers.Env("BOT_ENV", "test")
	testBotToken := helpers.Env("TEST_BOT_TOKEN", "")
	prodBotToken := helpers.Env("PROD_BOT_TOKEN", "")

	return &Config{
		Telegram: TelegramConfig{
			BotEnv:       botEnv,
			TestBotToken: testBotToken,
			ProdBotToken: prodBotToken,
			BotToken:     selectBotToken(botEnv, testBotToken, prodBotToken),
		},
		Iiko: IikoConfig{
			Host:     helpers.Env("IIKO_HOST", ""),
			Port:     helpers.EnvInt("IIKO_PORT", 443),
			Login:    helpers.Env("IIKO_LOGIN", ""),
			Password: helpers.Env("IIKO_PASSWORD", ""),
		},
		DatabaseURL: databaseURL(),
		Sync: SyncConfig{
			Interval: helpers.EnvDuration("SYNC_INTERVAL", 6*time.Hour),
		},
	}
}

func databaseURL() string {
	if value := helpers.Env("DATABASE_URL", ""); value != "" {
		return value
	}
	if value := helpers.Env("DB_DSN", ""); value != "" {
		return value
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		helpers.Env("POSTGRES_USER", "postgres"),
		helpers.Env("POSTGRES_PASSWORD", "postgres"),
		helpers.Env("POSTGRES_HOST", "localhost"),
		helpers.EnvInt("POSTGRES_PORT", 5432),
		helpers.Env("POSTGRES_DB", "bakery"),
		helpers.Env("POSTGRES_SSLMODE", "disable"),
	)
}

func selectBotToken(botEnv string, testToken string, prodToken string) string {
	switch botEnv {
	case "prod", "production":
		return prodToken
	default:
		return testToken
	}
}
