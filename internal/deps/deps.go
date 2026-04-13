package deps

import (
	"time"

	"bakery/internal/adapter/inbound/http"
	"bakery/internal/adapter/outbound/db"
	"bakery/internal/adapter/outbound/postgres"
	"bakery/internal/app"
	"bakery/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	queries        *db.Queries
	clientRepo     *postgres.ClientRepository
	kitchenRepo    *postgres.KitchenRepository
	userRepo       *postgres.UserRepository
	orderRepo      *postgres.OrderRepository
	productRepo    *postgres.ProductRepository
	authService    *app.AuthService
	kitchenService *app.KitchenService
	clientService  *app.ClientService
	orderService   *app.OrderService
	productService *app.ProductService
	server         *http.Server
}

type opt func(*Deps) error

func New(opts ...opt) (*Deps, error) {
	d := &Deps{}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func WithRepos(pool *pgxpool.Pool) opt {
	return func(d *Deps) error {
		d.queries = db.New(pool)
		d.clientRepo = postgres.NewClientRepository(d.queries)
		d.kitchenRepo = postgres.NewKitchenRepository(d.queries)
		d.userRepo = postgres.NewUserRepository(d.queries)
		d.orderRepo = postgres.NewOrderRepository(d.queries)
		d.productRepo = postgres.NewProductRepository(d.queries)
		return nil
	}
}

func WithServices(config *config.Config) opt {
	return func(d *Deps) error {
		d.authService = app.NewAuthService(d.userRepo, d.clientRepo, d.kitchenRepo, config.AuthSecret, 24*time.Hour)
		d.kitchenService = app.NewKitchenService(d.kitchenRepo)
		d.clientService = app.NewClientService(d.clientRepo)
		d.orderService = app.NewOrderService(d.orderRepo, d.productRepo, d.clientRepo, d.kitchenRepo)
		d.productService = app.NewProductService(d.productRepo)
		return nil
	}
}

func WithServer() opt {
	return func(d *Deps) error {
		d.server = http.NewServer(d.authService, d.kitchenService, d.clientService, d.orderService, d.productService)
		return nil
	}
}

func (d *Deps) Server() *http.Server {
	return d.server
}
