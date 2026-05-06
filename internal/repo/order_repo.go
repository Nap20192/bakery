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
	txRead  = &sql.TxOptions{Isolation: sql.LevelReadUncommitted, ReadOnly: true}
	txWrite = &sql.TxOptions{Isolation: sql.LevelSerializable}
)

type SQLiteOrderRepo struct {
	db *sql.DB
}

func NewSQLiteOrderRepo(db *sql.DB) domain.OrderRepository {
	return &SQLiteOrderRepo{db: db}
}

func OpenSQLite(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}
	if _, err := db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA read_uncommitted = 1;
	`); err != nil {
		return nil, fmt.Errorf("sqlite: pragma: %w", err)
	}
	return db, nil
}

func (r *SQLiteOrderRepo) nextNumber(tx *sql.Tx, now time.Time) (string, error) {
	dayKey := now.Format("02012006")

	if _, err := tx.Exec(
		`INSERT INTO order_counters(day, counter) VALUES(?, 0) ON CONFLICT(day) DO NOTHING`,
		dayKey,
	); err != nil {
		return "", err
	}

	var counter int
	if err := tx.QueryRow(
		`UPDATE order_counters SET counter = counter + 1 WHERE day = ? RETURNING counter`,
		dayKey,
	).Scan(&counter); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_ORDER_%04d", dayKey, counter), nil
}

func (r *SQLiteOrderRepo) Create(input domain.CreateOrderInput) (domain.Order, error) {
	now := input.Date
	if now.IsZero() {
		now = time.Now()
	}

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
		`INSERT INTO orders(number, location, created_at) VALUES(?, ?, ?)`,
		number,
		input.Location,
		now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return domain.Order{}, err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		return domain.Order{}, err
	}

	for _, item := range input.Items {
		if _, err = tx.Exec(
			`INSERT INTO order_items(order_id, product_name, quantity) VALUES(?, ?, ?)`,
			orderID,
			item.Product,
			item.Quantity,
		); err != nil {
			return domain.Order{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Order{}, err
	}

	return domain.Order{
		ID:        fmt.Sprintf("%d", orderID),
		Number:    number,
		Location:  input.Location,
		Items:     input.Items,
		CreatedAt: now,
	}, nil
}

func (r *SQLiteOrderRepo) GetByNumber(number string) (domain.Order, error) {
	tx, err := r.db.BeginTx(context.Background(), txRead)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.Query(`
		SELECT o.id, o.number, o.location, o.created_at, i.product_name, i.quantity
		FROM orders o
		LEFT JOIN order_items i ON i.order_id = o.id
		WHERE o.number = ?
		ORDER BY i.id`,
		number,
	)
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
	tx, err := r.db.BeginTx(context.Background(), txRead)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.Query(`
		SELECT o.id, o.number, o.location, o.created_at, i.product_name, i.quantity
		FROM (SELECT id, number, location, created_at FROM orders ORDER BY id DESC LIMIT ?) o
		LEFT JOIN order_items i ON i.order_id = o.id
		ORDER BY o.id DESC, i.id`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrderRows(rows)
}

func scanOrderRows(rows *sql.Rows) ([]domain.Order, error) {
	var orders []domain.Order
	index := map[string]int{}

	for rows.Next() {
		var (
			id, number, location, createdAt string
			productName                     sql.NullString
			quantity                        sql.NullInt64
		)
		if err := rows.Scan(&id, &number, &location, &createdAt, &productName, &quantity); err != nil {
			return nil, err
		}

		pos, exists := index[number]
		if !exists {
			created, _ := time.Parse(time.RFC3339, createdAt)
			orders = append(orders, domain.Order{
				ID:        id,
				Number:    number,
				Location:  location,
				CreatedAt: created,
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
