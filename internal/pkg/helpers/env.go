package helpers

import (
	"os"
	"strconv"
	"time"
)

func Env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func EnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func EnvInt(key string, fallback int) int {
	value := Env(key, "")
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return fallback
}

func EnvDuration(key string, fallback time.Duration) time.Duration {
	value := Env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
