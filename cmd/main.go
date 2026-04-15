package main

import (
	"log"
	"os"

	"bakery/internal/app"
	"bakery/internal/bot"
	"bakery/internal/repo"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN не задан")
	}

	productRepo := repo.NewJsonProductRepository("products.json")

	doughRepo, err := repo.NewJsonDoughRepository("daugh.json")
	if err != nil {
		log.Fatalf("dough repo: %v", err)
	}

	db, err := repo.OpenSQLite("bakery.db")
	if err != nil {
		log.Fatalf("sqlite: %v", err)
	}
	defer db.Close()

	orderRepo := repo.NewSQLiteOrderRepo(db)
	orderSvc := app.NewOrderService(productRepo)

	b, err := bot.New(token, orderSvc, orderRepo, productRepo, doughRepo)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	log.Println("Бот запущен")
	b.Start()
}
