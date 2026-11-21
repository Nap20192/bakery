package deps

import (
	"fmt"
	"strconv"

	"bakery/internal/config"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InfraDeps struct {
	config     *config.Config
	DB         *pgxpool.Pool
	iikoClient *iiko.Client
	queries    *sqlc.Queries
}

type infraOption func(*InfraDeps) error

func NewInfraDeps(opts ...infraOption) (*InfraDeps, error) {
	deps := &InfraDeps{}
	for _, opt := range opts {
		if err := opt(deps); err != nil {
			return nil, err
		}
	}
	return deps, nil
}

func WithConfig(cfg *config.Config) infraOption {
	return func(deps *InfraDeps) error {
		if cfg == nil {
			return fmt.Errorf("missing config")
		}
		deps.config = cfg
		return nil
	}
}

func WithPostgres(db *pgxpool.Pool) infraOption {
	return func(deps *InfraDeps) error {
		if deps.config == nil {
			return fmt.Errorf("missing dependencies for Postgres")
		}
		if db == nil {
			return fmt.Errorf("missing postgres db")
		}
		deps.DB = db
		return nil
	}
}

func WithRepositories() infraOption {
	return func(deps *InfraDeps) error {
		if deps.DB == nil {
			return fmt.Errorf("missing dependencies for repositories")
		}
		deps.queries = sqlc.New(deps.DB)
		return nil
	}
}

func WithIikoClient() infraOption {
	return func(deps *InfraDeps) error {
		if deps.config == nil {
			return fmt.Errorf("missing dependencies for iiko client")
		}
		cfg := deps.config.Iiko
		if cfg.Host == "" || cfg.Login == "" || cfg.Password == "" {
			return fmt.Errorf("IIKO_HOST, IIKO_LOGIN and IIKO_PASSWORD must be set")
		}
		client, err := iiko.NewClient(cfg.Login, cfg.Password, iiko.NewApi(cfg.Host, strconv.Itoa(cfg.Port)))
		if err != nil {
			return err
		}
		deps.iikoClient = client
		return nil
	}
}
