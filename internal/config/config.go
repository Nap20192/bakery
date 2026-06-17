package config

import (
	"fmt"
	"time"

	"bakery/internal/pkg/helpers"
)

type TelegramConfig struct {
	BotEnv         string
	TestBotToken   string
	ProdBotToken   string
	BotToken       string
	MiniAppURL     string
	WorkshopChatID int64
}

type LogConfig struct {
	Level  string
	Pretty bool
	Dir    string
}

type ServerConfig struct {
	Host            string
	Port            int
	AllowedOrigins  string
	ShutdownTimeout time.Duration
}

func (c ServerConfig) Addr() string {
	if c.Host == "" {
		return fmt.Sprintf(":%d", c.Port)
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type DatabaseConfig struct {
	URL      string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type MigrationConfig struct {
	Dir string
}

type AdminConfig struct {
	Username string
	Password string
}

type IikoConfig struct {
	Host     string
	Port     int
	Login    string
	Password string
}

type Config struct {
	Admin        AdminConfig
	Database     DatabaseConfig
	Iiko         IikoConfig
	Log          LogConfig
	Migration    MigrationConfig
	OrderCleanup OrderCleanupConfig
	RabbitMQ     RabbitMQConfig
	Server       ServerConfig
	Sync         SyncConfig
	Telegram     TelegramConfig
}

type RabbitMQConfig struct {
	URL string
}

type SyncConfig struct {
	Interval time.Duration
}

type OrderCleanupConfig struct {
	Interval  time.Duration
	Retention time.Duration
}

func New() *Config {
	botEnv := helpers.Env("BOT_ENV", "test")
	testBotToken := helpers.Env("TEST_BOT_TOKEN", "")
	prodBotToken := helpers.Env("PROD_BOT_TOKEN", "")
	botToken := helpers.Env("BOT_TOKEN", "")
	database := databaseConfig()

	return &Config{
		Admin: AdminConfig{
			Username: helpers.Env("ADMIN_USERNAME", "admin"),
			Password: helpers.Env("ADMIN_PASSWORD", ""),
		},
		Database: database,
		Telegram: TelegramConfig{
			BotEnv:       botEnv,
			TestBotToken: testBotToken,
			ProdBotToken: prodBotToken,
			BotToken:     selectBotToken(botEnv, testBotToken, prodBotToken, botToken),
			MiniAppURL:   helpers.Env("MINI_APP_URL", ""),
			WorkshopChatID: int64(helpers.EnvInt(
				"TELEGRAM_WORKSHOP_CHAT_ID",
				0,
			)),
		},
		Iiko: IikoConfig{
			Host:     helpers.Env("IIKO_HOST", ""),
			Port:     helpers.EnvInt("IIKO_PORT", 443),
			Login:    helpers.Env("IIKO_LOGIN", ""),
			Password: helpers.Env("IIKO_PASSWORD", ""),
		},
		Log: LogConfig{
			Level:  helpers.Env("LOG_LEVEL", "INFO"),
			Pretty: helpers.EnvBool("LOG_PRETTY", false),
			Dir:    helpers.Env("LOG_DIR", ""),
		},
		Migration: MigrationConfig{
			Dir: helpers.Env("MIGRATIONS_DIR", "migrations"),
		},
		Server: ServerConfig{
			Host:            helpers.Env("HTTP_HOST", ""),
			Port:            serverPort(),
			AllowedOrigins:  helpers.Env("HTTP_ALLOWED_ORIGINS", ""),
			ShutdownTimeout: helpers.EnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		RabbitMQ: RabbitMQConfig{
			URL: helpers.Env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		Sync: SyncConfig{
			Interval: helpers.EnvDuration("SYNC_INTERVAL", 6*time.Hour),
		},
		OrderCleanup: OrderCleanupConfig{
			Interval:  helpers.EnvDuration("ORDER_CLEANUP_INTERVAL", 24*time.Hour),
			Retention: helpers.EnvDuration("ORDER_RETENTION", 31*24*time.Hour),
		},
	}
}

func databaseConfig() DatabaseConfig {
	cfg := DatabaseConfig{
		Host:     helpers.Env("POSTGRES_HOST", "localhost"),
		Port:     helpers.EnvInt("POSTGRES_PORT", 5432),
		Name:     helpers.Env("POSTGRES_DB", "bakery"),
		User:     helpers.Env("POSTGRES_USER", "postgres"),
		Password: helpers.Env("POSTGRES_PASSWORD", "postgres"),
		SSLMode:  helpers.Env("POSTGRES_SSLMODE", "disable"),
	}
	if value := helpers.Env("DATABASE_URL", ""); value != "" {
		cfg.URL = value
		return cfg
	}
	cfg.URL = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
	return cfg
}

func serverPort() int {
	if port := helpers.EnvInt("PORT", 0); port > 0 {
		return port
	}
	return helpers.EnvInt("HTTP_PORT", 8080)
}

func selectBotToken(botEnv string, testToken string, prodToken string, fallbackToken string) string {
	switch botEnv {
	case "prod", "production":
		if prodToken != "" {
			return prodToken
		}
	default:
		if testToken != "" {
			return testToken
		}
	}
	return fallbackToken
}
