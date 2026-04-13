package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"bakery/internal/app"
	"bakery/internal/domain" // Твои структуры
	"bakery/internal/repo"

	"gopkg.in/telebot.v3"
)

var (
	// Список всех принятых заказов
	finalizedOrders []domain.Order
	// Сервис из твоего кода
	// Твой ID как менеджера
	managerID int64 = 123456789
)

func main() {
	// --- ГЕНЕРАЦИЯ ТЕСТОВЫХ ЗАКАЗОВ ---
	finalizedOrders = []domain.Order{
		{
			ID: "ЗАКАЗ-ПЕКАРНЯ-1",
			Items: []domain.OrderItem{
				{Product: "Булочка Улитка с корицей", Quantity: 15},
				{Product: "Булочка Улитка с маком", Quantity: 10},
			},
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID: "ЗАКАЗ-ПЕКАРНЯ-2",
			Items: []domain.OrderItem{
				{Product: "Шанежка с сыром", Quantity: 25},
				{Product: "Булочка Улитка с маком", Quantity: 5},
			},
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID: "СРОЧНЫЙ-ДОП",
			Items: []domain.OrderItem{
				{Product: "Булочка Улитка с корицей", Quantity: 50},
			},
			CreatedAt: time.Now(),
		},
	}
	// ----------------------------------
	orderRepo := repo.NewJsonProductRepository("./products.json")
	orderService := app.NewOrderService(orderRepo)
	// ... настройки бота ...
	// ВНИМАНИЕ: Обязательно смени токен в BotFather!
	pref := telebot.Settings{
		Token:  "<REMOVED_TELEGRAM_BOT_TOKEN>",
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}
	// --- МЕНЮ МЕНЕДЖЕРА ---
	b.Handle("/admin", func(c telebot.Context) error {

		m := &telebot.ReplyMarkup{}
		btnSelect := m.Data("🔍 Выбрать заказ", "adm_select")
		btnTotal := m.Data("📊 Общий итог", "adm_total")
		btnIngredients := m.Data("🍞 Расчет теста", "adm_calc_ingredients")

		m.Inline(m.Row(btnSelect), m.Row(btnTotal), m.Row(btnIngredients))
		return c.Send("🛠 Панель управления заказами:", m)
	})

	// 1. СПИСОК ЗАКАЗОВ ДЛЯ ВЫБОРА
	b.Handle(&telebot.InlineButton{Unique: "adm_select"}, func(c telebot.Context) error {
		if len(finalizedOrders) == 0 {
			return c.Edit("Заказов пока нет.")
		}

		m := &telebot.ReplyMarkup{}
		var rows []telebot.Row

		for i, ord := range finalizedOrders {
			// Создаем кнопку для каждого заказа
			btn := m.Data(fmt.Sprintf("📦 %s", ord.ID), "view_ord", strconv.Itoa(i))
			rows = append(rows, m.Row(btn))
		}

		btnBack := m.Data("⬅️ Назад", "adm_main")
		rows = append(rows, m.Row(btnBack))

		m.Inline(rows...)
		return c.Edit("Выберите заказ для просмотра деталей:", m)
	})

	// 2. ПРОСМОТР КОНКРЕТНОГО ЗАКАЗА
	b.Handle(&telebot.InlineButton{Unique: "view_ord"}, func(c telebot.Context) error {
		idx, _ := strconv.Atoi(c.Data())
		if idx >= len(finalizedOrders) {
			return c.Respond(&telebot.CallbackResponse{Text: "Заказ не найден"})
		}

		ord := finalizedOrders[idx]

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📝 **Детали заказа %s**\n\n", ord.ID))
		for _, item := range ord.Items {
			sb.WriteString(fmt.Sprintf("• %s: %d шт.\n", item.Product, item.Quantity))
		}

		m := &telebot.ReplyMarkup{}
		btnBack := m.Data("⬅️ К списку", "adm_select")
		m.Inline(m.Row(btnBack))

		return c.Edit(sb.String(), m, telebot.ModeMarkdown)
	})

	// 3. ОБЩИЙ ИТОГ (CombineOrders)
	b.Handle(&telebot.InlineButton{Unique: "adm_total"}, func(c telebot.Context) error {
		if len(finalizedOrders) == 0 {
			return c.Edit("Нет данных для объединения.")
		}

		// Вызываем твой сервис
		combined := orderService.CombineOrders(finalizedOrders)

		var sb strings.Builder
		sb.WriteString("📊 **Общий план выпечки:**\n\n")
		for _, item := range combined.Items {
			sb.WriteString(fmt.Sprintf("✅ %s: %d шт.\n", item.Product, item.Quantity))
		}

		m := &telebot.ReplyMarkup{}
		m.Inline(m.Row(m.Data("⬅️ В меню", "adm_main")))
		return c.Edit(sb.String(), m, telebot.ModeMarkdown)
	})

	// 4. РАСЧЕТ ИНГРЕДИЕНТОВ (CalculateTotalIngredient)
	b.Handle(&telebot.InlineButton{Unique: "adm_calc_ingredients"}, func(c telebot.Context) error {
		if len(finalizedOrders) == 0 {
			return c.Edit("Заказов нет — считать нечего.")
		}

		combined := orderService.CombineOrders(finalizedOrders)

		// Пример: считаем сколько нужно "Тесто Сдобное" на весь объем
		target := "Тесто Сдобное п/ф"
		res, err := orderService.CalculateTotalIngredient(combined, target)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}

		text := fmt.Sprintf("🥖 **Расчет по позиции: %s**\n\n", target)
		text += fmt.Sprintf("💰 Итого нужно замесить: **%.2f кг**\n\n", res.Total)
		text += "Распределение по изделиям:\n"
		for _, p := range res.Products {
			text += fmt.Sprintf("— %s: %d шт.\n", p.Product, p.Quantity)
		}

		m := &telebot.ReplyMarkup{}
		m.Inline(m.Row(m.Data("⬅️ В меню", "adm_main")))
		return c.Edit(text, m, telebot.ModeMarkdown)
	})

	// Возврат в главное админ-меню
	b.Handle(&telebot.InlineButton{Unique: "adm_main"}, func(c telebot.Context) error {
		m := &telebot.ReplyMarkup{}
		btnSelect := m.Data("🔍 Выбрать заказ", "adm_select")
		btnTotal := m.Data("📊 Общий итог", "adm_total")
		btnIngredients := m.Data("🍞 Расчет теста", "adm_calc_ingredients")
		m.Inline(m.Row(btnSelect), m.Row(btnTotal, btnIngredients))
		return c.Edit("🛠 Панель управления заказами:", m)
	})

	b.Start()
}
