package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bakery/internal/config"
	"bakery/internal/deps"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	_ = godotenv.Load()

	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	poolCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(poolCtx, cfg.Postgres.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(poolCtx); err != nil {
		return err
	}

	d, err := deps.New(
		deps.WithRepos(pool),
		deps.WithServices(cfg),
		deps.WithServer(),
	)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           d.Server(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Printf("http: listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Printf("http: shutting down")
		return srv.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
