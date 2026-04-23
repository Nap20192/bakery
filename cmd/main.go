package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"bakery/internal/app"
	"bakery/internal/bot"
	"bakery/internal/config"
	"bakery/internal/repo"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	cfg := config.New()

	if cfg.Telegram.BotToken == "" {
		log.Fatal("BOT_TOKEN не задан")
	}

	productRepo := repo.NewJsonProductRepository("products.json")

	doughRepo, err := repo.NewJsonDoughRepository("daugh.json")
	if err != nil {
		log.Fatalf("dough repo: %v", err)
	}

	db, err := repo.OpenSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("sqlite: %v", err)
	}

	orderRepo := repo.NewSQLiteOrderRepo(db)
	orderSvc := app.NewOrderService(productRepo)

	b, err := bot.New(cfg.Telegram.BotToken, orderSvc, orderRepo, productRepo, doughRepo)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		log.Printf("Получен сигнал %s, останавливаем бот...", sig)
		b.Stop()
	}()

	log.Println("Бот запущен")
	b.Start()

	log.Println("Закрываем БД...")
	if err := db.Close(); err != nil {
		log.Printf("db.Close: %v", err)
	}
	log.Println("Остановлен.")
}
