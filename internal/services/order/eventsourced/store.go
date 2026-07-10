package eventsourced

import (
	stdsql "database/sql"
	"fmt"

	"github.com/hallgren/eventsourcing/aggregate"
	essql "github.com/hallgren/eventsourcing/eventstore/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" для event store
)

// RegisterAggregates регистрирует event-sourced агрегаты в глобальном
// реестре библиотеки. Вызывается один раз на старте бинаря (composition
// root) до первого Save/Load.
func RegisterAggregates() {
	aggregate.Register(&Order{})
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
