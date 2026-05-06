package deps

import (
	"database/sql"
	"fmt"
	"strconv"

	"bakery/internal/config"
	"bakery/internal/iiko"
	sqlcrepo "bakery/internal/repo/sqlc"
)

type InfraDeps struct {
	Config      *config.Config
	DB          *sql.DB
	IikoClient  *iiko.Client
	IikoQueries *sqlcrepo.Queries
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
		deps.Config = cfg
		return nil
	}
}

func WithSQLite() infraOption {
	return func(deps *InfraDeps) error {
		if deps.Config == nil {
			return fmt.Errorf("missing dependencies for SQLite")
		}
		db, err := repo.OpenSQLite(deps.Config.DBPath)
		if err != nil {
			return err
		}
		deps.DB = db
		return nil
	}
}

func WithRepositories() infraOption {
	return func(deps *InfraDeps) error {
		if deps.Config == nil || deps.DB == nil {
			return fmt.Errorf("missing dependencies for repositories")
		}
		productRepo := repo.NewJsonProductRepository("all.json")
		doughRepo, err := repo.NewJsonDoughRepository("daugh.json")
		if err != nil {
			return err
		}
		deps.ProductRepo = productRepo
		deps.DoughRepo = doughRepo
		deps.OrderRepo = repo.NewSQLiteOrderRepo(deps.DB)
		deps.IikoQueries = sqlcrepo.New(deps.DB)
		return nil
	}
}

func WithIikoClient() infraOption {
	return func(deps *InfraDeps) error {
		if deps.Config == nil {
			return fmt.Errorf("missing dependencies for iiko client")
		}
		cfg := deps.Config.Iiko
		if cfg.Host == "" || cfg.Login == "" || cfg.Password == "" {
			return fmt.Errorf("IIKO_HOST, IIKO_LOGIN and IIKO_PASSWORD must be set")
		}
		client, err := iiko.NewClient(cfg.Login, cfg.Password, iiko.NewApi(cfg.Host, strconv.Itoa(cfg.Port)))
		if err != nil {
			return err
		}
		deps.IikoClient = client
		return nil
	}
}
