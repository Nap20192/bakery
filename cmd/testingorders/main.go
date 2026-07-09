// Command testingorders — dev-утилита: удаляет ВСЕ заказы (вместе с
// отработками, счётчиками и outbox) и генерирует свежие тестовые заявки —
// по одной на каждый магазин × тип заявки × день недели, со случайными
// позициями из каталога своего типа.
//
// Заказы пишутся напрямую в БД (без usecase-слоя) сознательно: тестовые
// данные не должны порождать события в outbox и уведомления бота. Номера
// строятся доменной логикой — формат один на всё приложение.
//
// Запуск: go run ./cmd/testingorders -yes
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"time"

	"bakery/internal/config"
	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/pkg/helpers"
	orderdomain "bakery/internal/services/order/domain"
	"bakery/pkg/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const days = 7

type shop struct {
	ID   int64
	Code string
	Name string
}

type category struct {
	ID     int64
	Letter string
	Name   string
}

type dish struct {
	Name      string
	ProductID *string
}

func main() {
	yes := flag.Bool("yes", false, "подтвердить удаление всех заказов")
	flag.Parse()

	_ = config.LoadDotenv()
	cfg := config.New()
	log, err := logger.InitLogger(cfg.Log.Level, cfg.Log.Pretty, "")
	if err != nil {
		panic(err)
	}
	slog.SetDefault(log)

	if !*yes {
		fmt.Println("testingorders удаляет ВСЕ заказы, отработки и счётчики, затем создаёт тестовые заявки.")
		fmt.Println("Запустите с флагом -yes, чтобы подтвердить: go run ./cmd/testingorders -yes")
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := outbounddb.OpenPostgres(ctx, cfg.Database.URL)
	if err != nil {
		log.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer helpers.ClosePool(db)

	if err := run(ctx, db); err != nil {
		log.Error("testingorders failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, db *pgxpool.Pool) error {
	// 1. Полная очистка: отработки первыми (FK на заказы), затем заказы,
	// счётчики номеров и невыгребенный outbox.
	for _, table := range []string{"production_sheets", "orders", "order_counters", "order_outbox"} {
		if _, err := db.Exec(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	slog.Info("cleared orders, production sheets, counters, outbox")

	shops, err := loadShops(ctx, db)
	if err != nil {
		return err
	}
	workshopID, err := loadWorkshopID(ctx, db)
	if err != nil {
		return err
	}
	categories, err := loadCategories(ctx, db)
	if err != nil {
		return err
	}

	domain := orderdomain.NewOrderService()
	now := time.Now().UTC()
	counterDay := domain.OrderCounterDay(now)

	created := 0
	for _, s := range shops {
		for _, c := range categories {
			dishes, err := loadDishes(ctx, db, c.ID)
			if err != nil {
				return err
			}
			if len(dishes) == 0 {
				slog.Warn("category has no dishes, skipping", "category", c.Name)
				continue
			}
			for offset := 0; offset < days; offset++ {
				counter, err := nextCounter(ctx, db, counterDay, s.ID, c.ID)
				if err != nil {
					return err
				}
				number := domain.BuildOrderNumber(s.Code, s.Name, c.Letter, now, counter)
				fulfillment := now.AddDate(0, 0, offset)
				if err := insertOrder(ctx, db, number, s, workshopID, c.ID, fulfillment, dishes); err != nil {
					return fmt.Errorf("insert order %s: %w", number, err)
				}
				created++
			}
		}
	}
	slog.Info("test orders created", "orders", created, "shops", len(shops), "categories", len(categories), "days", days)
	return nil
}

func loadShops(ctx context.Context, db *pgxpool.Pool) ([]shop, error) {
	rows, err := db.Query(ctx, "SELECT id, code, name FROM departments WHERE type = 'shop' ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("load shops: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[shop])
}

func loadWorkshopID(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	var id int64
	if err := db.QueryRow(ctx, "SELECT id FROM departments WHERE code = 'pekari'").Scan(&id); err != nil {
		return 0, fmt.Errorf("load workshop: %w", err)
	}
	return id, nil
}

func loadCategories(ctx context.Context, db *pgxpool.Pool) ([]category, error) {
	rows, err := db.Query(ctx, "SELECT id, letter, name FROM order_categories ORDER BY sort_order, id")
	if err != nil {
		return nil, fmt.Errorf("load categories: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[category])
}

func loadDishes(ctx context.Context, db *pgxpool.Pool, categoryID int64) ([]dish, error) {
	rows, err := db.Query(ctx, `
		SELECT dc.name, p.id
		FROM dish_catalog dc
		LEFT JOIN iiko_products p ON p.code = dc.code AND p.type = 'DISH'
		WHERE dc.category_id = $1`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("load dishes: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[dish])
}

func nextCounter(ctx context.Context, db *pgxpool.Pool, day string, shopID, categoryID int64) (int64, error) {
	if _, err := db.Exec(ctx, `
		INSERT INTO order_counters(day, department_id, category_id, counter)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (day, department_id, category_id) DO NOTHING`, day, shopID, categoryID); err != nil {
		return 0, fmt.Errorf("init counter: %w", err)
	}
	var counter int64
	if err := db.QueryRow(ctx, `
		UPDATE order_counters SET counter = counter + 1
		WHERE day = $1 AND department_id = $2 AND category_id = $3
		RETURNING counter`, day, shopID, categoryID).Scan(&counter); err != nil {
		return 0, fmt.Errorf("increment counter: %w", err)
	}
	return counter, nil
}

func insertOrder(ctx context.Context, db *pgxpool.Pool, number string, s shop, workshopID, categoryID int64, fulfillment time.Time, dishes []dish) error {
	var orderID int64
	err := db.QueryRow(ctx, `
		INSERT INTO orders (number, location, from_department_id, to_department_id, category_id,
		                    created_at, fulfillment_date, created_by_username)
		VALUES ($1, $2, $3, $4, $5, now(), $6, 'admin')
		RETURNING id`,
		number, s.Name, s.ID, workshopID, categoryID, fulfillment.Format("2006-01-02")).Scan(&orderID)
	if err != nil {
		return err
	}

	// 3–5 случайных блюд, количества 5–30, у ~трети позиций резерв 1–5.
	// Тестовые данные — криптостойкость генератора не нужна.
	picked := rand.Perm(len(dishes))[:min(3+rand.IntN(3), len(dishes))] //nolint:gosec
	for _, index := range picked {
		d := dishes[index]
		reserved := 0
		if rand.IntN(10) < 3 { //nolint:gosec
			reserved = 1 + rand.IntN(5) //nolint:gosec
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO order_items (order_id, iiko_product_id, product_name, quantity, reserved_quantity)
			VALUES ($1, $2, $3, $4, $5)`,
			orderID, d.ProductID, d.Name, 5+rand.IntN(26), reserved); err != nil { //nolint:gosec
			return err
		}
	}
	return nil
}
