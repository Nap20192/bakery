package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"bakery/internal/adapter/outbound/postgres"
	httpserver "bakery/internal/adapter/inbound/http"
	"bakery/internal/app"
	"bakery/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("postgres ping: %v", err)
	}
	log.Println("connected to postgres")

	queries := postgres.New(pool)

	kitchenSvc := app.NewKitchenService(queries)
	clientSvc := app.NewClientService(queries)
	orderSvc := app.NewOrderService(queries)

	srv := httpserver.NewServer(kitchenSvc, clientSvc, orderSvc)

	addr := fmt.Sprintf(":%s", env("HTTP_PORT", "8080"))
	httpSrv := &http.Server{Addr: addr, Handler: srv}

	go func() {
		log.Printf("http server listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	if err := httpSrv.Shutdown(context.Background()); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
