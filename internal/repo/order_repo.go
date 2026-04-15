package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bakery/internal/domain"

	_ "modernc.org/sqlite"
)

var (
	// txRead — читаем «грязно»: не ждём чужих незакоммиченных записей.
	// В SQLite это работает через shared-cache + PRAGMA read_uncommitted.
	txRead = &sql.TxOptions{Isolation: sql.LevelReadUncommitted, ReadOnly: true}

	// txWrite — полная сериализация для записи: счётчик ORDER_NNNN должен быть атомарным.
	txWrite = &sql.TxOptions{Isolation: sql.LevelSerializable}
)

type SQLiteOrderRepo struct {
	db *sql.DB
}

func NewSQLiteOrderRepo(db *sql.DB) domain.OrderRepository {
	return &SQLiteOrderRepo{db: db}
}

func OpenSQLite(path string) (*sql.DB, error) {
	// shared cache обязателен для PRAGMA read_uncommitted
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// shared cache требует единственного пула соединений
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}
	if _, err := db.Exec(`
		PRAGMA journal_mode   = WAL;
		PRAGMA synchronous    = NORMAL;
		PRAGMA read_uncommitted = 1;
	`); err != nil {
		return nil, fmt.Errorf("sqlite: pragma: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("sqlite: migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS order_counters (
			day  TEXT PRIMARY KEY,
			counter INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS orders (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			number     TEXT    NOT NULL UNIQUE,
			created_at TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS order_items (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id     INTEGER NOT NULL REFERENCES orders(id),
			product_name TEXT    NOT NULL,
			quantity     INTEGER NOT NULL
		);
	`)
	return err
}

// nextNumber инкрементирует счётчик дня и возвращает номер вида DDMMYYYY_ORDER_NNNN.
func (r *SQLiteOrderRepo) nextNumber(tx *sql.Tx, now time.Time) (string, error) {
	dayKey := now.Format("02012006") // DDMMYYYY

	_, err := tx.Exec(
		`INSERT INTO order_counters(day, counter) VALUES(?, 0) ON CONFLICT(day) DO NOTHING`,
		dayKey,
	)
	if err != nil {
		return "", err
	}

	var counter int
	err = tx.QueryRow(
		`UPDATE order_counters SET counter = counter + 1 WHERE day = ? RETURNING counter`,
		dayKey,
	).Scan(&counter)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_ORDER_%04d", dayKey, counter), nil
}

func (r *SQLiteOrderRepo) Create(items []domain.OrderItem) (domain.Order, error) {
	now := time.Now()

	// LevelSerializable — счётчик ORDER_NNNN атомарен, дублей быть не должно
	tx, err := r.db.BeginTx(context.Background(), txWrite)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	number, err := r.nextNumber(tx, now)
	if err != nil {
		return domain.Order{}, fmt.Errorf("order number: %w", err)
	}

	res, err := tx.Exec(
		`INSERT INTO orders(number, created_at) VALUES(?, ?)`,
		number, now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return domain.Order{}, err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		return domain.Order{}, err
	}

	for _, item := range items {
		if _, err = tx.Exec(
			`INSERT INTO order_items(order_id, product_name, quantity) VALUES(?, ?, ?)`,
			orderID, item.Product, item.Quantity,
		); err != nil {
			return domain.Order{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return domain.Order{}, err
	}

	return domain.Order{
		ID:        fmt.Sprintf("%d", orderID),
		Number:    number,
		Items:     items,
		CreatedAt: now,
	}, nil
}

func (r *SQLiteOrderRepo) GetByNumber(number string) (domain.Order, error) {
	// LevelReadUncommitted — не ждём завершения параллельных записей
	tx, err := r.db.BeginTx(context.Background(), txRead)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.Query(`
		SELECT o.id, o.number, o.created_at,
		       i.product_name, i.quantity
		FROM orders o
		LEFT JOIN order_items i ON i.order_id = o.id
		WHERE o.number = ?
		ORDER BY i.id`, number)
	if err != nil {
		return domain.Order{}, err
	}
	defer rows.Close()

	orders, err := scanOrderRows(rows)
	if err != nil {
		return domain.Order{}, err
	}
	if len(orders) == 0 {
		return domain.Order{}, fmt.Errorf("заказ %q: %w", number, domain.ErrNotFound)
	}
	return orders[0], nil
}

func (r *SQLiteOrderRepo) List(limit int) ([]domain.Order, error) {
	// LevelReadUncommitted — список заказов читаем без ожидания блокировок
	tx, err := r.db.BeginTx(context.Background(), txRead)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.Query(`
		SELECT o.id, o.number, o.created_at,
		       i.product_name, i.quantity
		FROM (SELECT id, number, created_at FROM orders ORDER BY id DESC LIMIT ?) o
		LEFT JOIN order_items i ON i.order_id = o.id
		ORDER BY o.id DESC, i.id`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrderRows(rows)
}

// scanOrderRows читает результат JOIN-запроса (orders + order_items) и собирает срез Order.
func scanOrderRows(rows *sql.Rows) ([]domain.Order, error) {
	var orders []domain.Order
	index := map[string]int{} // number → позиция в orders

	for rows.Next() {
		var (
			id, number, createdAt string
			productName           sql.NullString
			quantity              sql.NullInt64
		)
		if err := rows.Scan(&id, &number, &createdAt, &productName, &quantity); err != nil {
			return nil, err
		}

		pos, exists := index[number]
		if !exists {
			t, _ := time.Parse(time.RFC3339, createdAt)
			orders = append(orders, domain.Order{
				ID:        id,
				Number:    number,
				CreatedAt: t,
			})
			pos = len(orders) - 1
			index[number] = pos
		}

		if productName.Valid {
			orders[pos].Items = append(orders[pos].Items, domain.OrderItem{
				Product:  productName.String,
				Quantity: int(quantity.Int64),
			})
		}
	}
	return orders, rows.Err()
}
