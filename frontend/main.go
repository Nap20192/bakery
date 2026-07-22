package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bakery/frontend/internal/application"
	"bakery/frontend/internal/backend"
	"bakery/frontend/internal/web"
	"bakery/internal/config"
)

func main() {
	if err := config.LoadDotenv(); err != nil {
		slog.Error("load dotenv", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	apiURL := env("BACKEND_URL", "http://127.0.0.1:8080")
	client := backend.New(apiURL, 20*time.Second)
	handler, err := web.New(application.NewQueries(client), application.NewCommands(client), logger)
	if err != nil {
		logger.Error("initialize frontend", "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              env("FRONTEND_ADDR", ":5173"),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		logger.Info("frontend started", "addr", httpServer.Addr, "backend_url", apiURL)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("serve frontend", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown frontend", "error", err)
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
