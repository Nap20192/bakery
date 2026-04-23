package config

import (
	"os"
	"strconv"
)

type TelegramConfig struct {
	BotToken string
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
}

func New() *Config {
	return &Config{
		Telegram: TelegramConfig{
			BotToken: getEnv("BOT_TOKEN", ""),
		},
		Iiko: IikoConfig{
			Host:     getEnv("IIKO_HOST", ""),
			Port:     getEnvAsInt("IIKO_PORT", 443),
			Login:    getEnv("IIKO_LOGIN", ""),
			Password: getEnv("IIKO_PASSWORD", ""),
		},
		DBPath:   getEnv("DB_PATH", "bakery.db"),
		HTTPPort: getEnvAsInt("HTTP_PORT", 8080),
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
