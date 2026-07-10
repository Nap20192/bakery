package eventsourced

import (
	stdsql "database/sql"
	"fmt"
	"sync"

	"github.com/hallgren/eventsourcing/aggregate"
	essql "github.com/hallgren/eventsourcing/eventstore/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" для event store
)

var registerOnce sync.Once

// RegisterAggregates регистрирует event-sourced агрегаты в глобальном
// реестре библиотеки до первого Save/Load. Повторные вызовы безопасны —
// регистрация выполняется один раз на процесс.
func RegisterAggregates() {
	registerOnce.Do(func() {
		aggregate.Register(&Order{})
	})
}

// NewPostgresStore открывает event store поверх Postgres. Библиотека ходит
// через database/sql, поэтому рядом с pgxpool открывается отдельное
// соединение через pgx-stdlib на тот же DSN. Схему таблицы events создаёт
// goose-миграция (00025) — здесь только подключение.
//
// Закрывать в обратном порядке: сначала store.Close(), затем db.Close().
func NewPostgresStore(dsn string) (*essql.Postgres, *stdsql.DB, error) {
	db, err := stdsql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open event store db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping event store db: %w", err)
	}
	store, err := essql.NewPostgres(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("create postgres event store: %w", err)
	}
	return store, db, nil
}
