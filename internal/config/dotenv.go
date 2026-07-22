package config

import "github.com/joho/godotenv"

// LoadDotenv loads .env from the working directory for local runs. A missing
// file is not an error: under Docker and on Railway the variables come from the
// environment itself. Shared by all entrypoints (worker, bot, frontend).
func LoadDotenv() {
	_ = godotenv.Load()
}
