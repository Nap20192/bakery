package config

import (
	"os"
	"strconv"
	"time"
)

type TelegramConfig struct {
	OrderBotToken string
	AdminBotToken string
}

type IikoConfig struct {
	Host     string
	Port     int
	Login    string
	Password string
}

type Config struct {
	Telegram TelegramConfig
	Iiko     IikoConfig
	DBPath   string
	HTTPPort int
	Sync     SyncConfig
}

type SyncConfig struct {
	Interval time.Duration
	DateFrom string
	DateTo   string
}

func New() *Config {
	return &Config{
		Telegram: TelegramConfig{
			OrderBotToken: getEnv("ORDER_BOT_TOKEN", getEnv("BOT_TOKEN", "")),
			AdminBotToken: getEnv("ADMIN_BOT_TOKEN", getEnv("VIEW_BOT_TOKEN", "")),
		},
		Iiko: IikoConfig{
			Host:     getEnv("IIKO_HOST", ""),
			Port:     getEnvAsInt("IIKO_PORT", 443),
			Login:    getEnv("IIKO_LOGIN", ""),
			Password: getEnv("IIKO_PASSWORD", ""),
		},
		DBPath:   getEnv("DB_PATH", "bakery.db"),
		HTTPPort: getEnvAsInt("HTTP_PORT", 8080),
		Sync: SyncConfig{
			Interval: getEnvAsDuration("SYNC_INTERVAL", 6*time.Hour),
			DateFrom: getEnv("SYNC_DATE_FROM", time.Now().Format("2006-01-02")),
			DateTo:   getEnv("SYNC_DATE_TO", time.Now().Format("2006-01-02")),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	valueStr := getEnv(name, "")
	if valueStr == "" {
		return defaultVal
	}
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultVal
}
