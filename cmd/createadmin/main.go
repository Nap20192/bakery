package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	"bakery/internal/app"
	"bakery/internal/config"
	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
	"bakery/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	log, err := logger.InitLogger(getEnv("LOG_LEVEL", "INFO"), getEnvBool("LOG_PRETTY", true), getEnv("LOG_DIR", ""))
	if err != nil {
		panic(err)
	}
	slog.SetDefault(log)

	cfg := config.New()
	username := flag.String("username", getEnv("ADMIN_USERNAME", "admin"), "admin username")
	password := flag.String("password", getEnv("ADMIN_PASSWORD", ""), "admin password")
	flag.Parse()

	if *password == "" {
		log.Error("admin password is required", "flag", "-password", "env", "ADMIN_PASSWORD")
		os.Exit(1)
	}

	db, err := openSQLite(cfg.DBPath)
	if err != nil {
		log.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer closeDB(log, db)

	authSvc := app.NewAuthService(sqlc.New(db))
	user, err := authSvc.CreateUserWithPassword(context.Background(), domain.PasswordAuthUserInput{
		Username: *username,
		Password: *password,
		Role:     domain.RoleAdmin,
	})
	if err != nil {
		log.Error("create admin failed", "username", *username, "error", err)
		os.Exit(1)
	}

	log.Info("admin created", "username", user.Username, "role", user.Role)
}

func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared&mode=rwc", path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func closeDB(log *slog.Logger, db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		log.Error("close db failed", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
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
