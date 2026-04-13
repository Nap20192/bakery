package main

import (
	"bakery/internal/iiko"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var (
	IIKO_HOST     = "pekarnya-gagarina.iiko.it"
	IIKO_PORT     = "443"
	IIKO_LOGIN    = "Nikolay"
	IIKO_PASSWORD = "3333"
)

func main() {
	api := iiko.NewApi(IIKO_HOST, IIKO_PORT)
	client, err := iiko.NewClient(IIKO_LOGIN, IIKO_PASSWORD, api)
	if err != nil {
		panic(err)
	}

	err = client.Auth()
	if err != nil {
		panic(err)
	}

	defer func() {
		err := client.Logout()
		if err != nil {
			panic(err)
		}
	}()
	dateFrom := "2024-01-01"
	dateTo := "2026-12-31" // Укажи актуальный диапазон

	charts, err := client.AssemblyChartsGetAll(dateFrom, dateTo, false, true)
	if err != nil {
		log.Fatalf("Ошибка получения техкарт: %v", err)
	}

	// 4. Преобразуем в JSON с отступами (для читаемости)
	jsonData, err := json.MarshalIndent(charts, "", "  ")
	if err != nil {
		log.Fatalf("Ошибка маршалинга в JSON: %v", err)
	}

	// 5. Сохраняем в файл
	fileName := "all_charts.json"
	err = os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		log.Fatalf("Ошибка записи в файл: %v", err)
	}

	fmt.Printf("Успешно! Все техкарты (%d шт.) сохранены в файл %s\n", len(charts.AssemblyCharts), fileName)

}
