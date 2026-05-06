package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"bakery/internal/config"
	"bakery/internal/deps"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	infra, err := deps.NewInfraDeps(
		deps.WithConfig(config.New()),
		deps.WithSQLite(),
		deps.WithRepositories(),
		deps.WithIikoClient(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer closeDB(infra)

	appDeps, err := deps.NewAppDeps(
		deps.WithOrderService(infra),
		deps.WithSyncService(infra),
		deps.WithAdminBot(infra),
	)
	if err != nil {
		log.Fatal(err)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Println("adminbot запущен")
		appDeps.AdminBot.Start()
		return nil
	})
	group.Go(func() error {
		<-groupCtx.Done()
		appDeps.AdminBot.Stop()
		return nil
	})
	group.Go(func() error {
		return appDeps.SyncService.Run(groupCtx)
	})

	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}
}

func closeDB(infra *deps.InfraDeps) {
	if infra == nil || infra.DB == nil {
		return
	}
	log.Println("Закрываем БД...")
	if err := infra.DB.Close(); err != nil {
		log.Printf("db.Close: %v", err)
	}
	log.Println("Остановлен.")
}
