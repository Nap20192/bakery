package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"bakery/internal/app"
	"bakery/internal/config"
	accessdomain "bakery/internal/domain/access"
	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/dbmigrate"
	"bakery/internal/pkg/helpers"
	"bakery/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	log, err := logger.InitLogger(helpers.Env("LOG_LEVEL", "INFO"), helpers.EnvBool("LOG_PRETTY", true), helpers.Env("LOG_DIR", ""))
	if err != nil {
		panic(err)
	}
	slog.SetDefault(log)

	cfg := config.New()
	username := flag.String("username", helpers.Env("ADMIN_USERNAME", "admin"), "admin username")
	password := flag.String("password", helpers.Env("ADMIN_PASSWORD", ""), "admin password")
	flag.Parse()

	if *password == "" {
		log.Error("admin password is required", "flag", "-password", "env", "ADMIN_PASSWORD")
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := outbounddb.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer helpers.ClosePool(db)
	if err := dbmigrate.ApplyInitialSchema(ctx, db, log); err != nil {
		log.Error("apply db schema failed", "error", err)
		os.Exit(1)
	}

	authSvc := app.NewAuthService(sqlc.New(db))
	user, err := authSvc.CreateUserWithPassword(ctx, accessdomain.PasswordAuthUserInput{
		Username: *username,
		Password: *password,
		Role:     accessdomain.RoleAdmin,
	})
	if err != nil {
		log.Error("create admin failed", "username", *username, "error", err)
		os.Exit(1)
	}

	log.Info("admin created", "username", user.Username, "role", user.Role)
}
