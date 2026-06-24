// Package orderuc is the application (use-case) layer of the order service.
//
// It owns the service's boundary contract (UseCase) and the ports it depends
// on (Repository). Following the dependency-inversion
// principle, this inner layer declares the interfaces it needs and never
// imports infrastructure; the infra adapters (infra/repo) depend on this
// package and implement these ports. Wiring happens in the composition root.
package orderuc

import (
	"context"
	"time"

	orderdomain "bakery/internal/services/order/domain"
)

// UseCase is the boundary the delivery layer (bot, HTTP API) talks to.
type UseCase interface {
	CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error)
	UpdateOrder(ctx context.Context, input UpdateOrderInput) (orderdomain.Order, error)
	GetOrderByNumber(ctx context.Context, number string) (orderdomain.Order, error)
	ListOrders(ctx context.Context, input ListOrdersInput) (ListOrdersResult, error)
	SetOrderFavorite(ctx context.Context, number string, favorite bool) (orderdomain.Order, error)
	ValidateBulkOrder(ctx context.Context, order string) orderdomain.BulkOrderValidationResult
	ListDishCatalog(ctx context.Context) ([]orderdomain.DishCatalogItem, error)
	AddDishCatalogItem(ctx context.Context, input orderdomain.DishCatalogItem) (orderdomain.DishCatalogItem, error)
	UpdateDishCatalogItem(ctx context.Context, code string, input orderdomain.DishCatalogItem) (orderdomain.DishCatalogItem, error)
	DeleteDishCatalogItem(ctx context.Context, code string) error
	ListOrderTemplates(ctx context.Context) ([]orderdomain.OrderTemplate, error)
	CombinedOrderTemplate(ctx context.Context) (string, error)
	GetOrderTemplate(ctx context.Context, theme string) (orderdomain.OrderTemplate, error)
	GetTemplate(ctx context.Context) (string, error)
	EnsureDefaultOrderTemplates(ctx context.Context, path string) (EnsureDefaultTemplatesResult, error)
	RunCleanupTicker(ctx context.Context, interval, retention time.Duration) error
	DeleteOrdersOlderThan(ctx context.Context, now time.Time, retention time.Duration) (int64, error)
}

// Repository is the persistence port. The infra/repo adapter implements it over
// the database; the use case depends only on this abstraction.
type Repository interface {
	CreateOrder(ctx context.Context, input CreateOrderRepositoryInput) (orderdomain.Order, error)
	UpdateOrder(ctx context.Context, input UpdateOrderRepositoryInput) (orderdomain.Order, error)
	GetOrderByNumber(ctx context.Context, number string) (orderdomain.Order, error)
	ListOrders(ctx context.Context, input ListOrdersInput) (ListOrdersResult, error)
	SetOrderFavorite(ctx context.Context, number string, favorite bool) (orderdomain.Order, error)
	GetDepartmentByID(ctx context.Context, id int64) (Department, error)
	DishExistsByCode(ctx context.Context, code string) (bool, error)
	ResolveDishCatalogItem(ctx context.Context, name string) (DishCatalogItem, error)
	ListDishCatalog(ctx context.Context) ([]DishCatalogItem, error)
	UpsertDishCatalogItem(ctx context.Context, item DishCatalogItem) error
	UpdateDishCatalogItem(ctx context.Context, code string, item DishCatalogItem) (DishCatalogItem, error)
	DeleteDishCatalogItem(ctx context.Context, code string) error
	DeleteOrdersOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

// Department is the persistence view of a department needed to create an order.
type Department struct {
	ID   int64
	Code string
	Name string
	Type string
}

// DishCatalogItem is the persistence view of a dish catalog row.
type DishCatalogItem struct {
	Code      string
	Name      string
	Theme     string
	SortOrder int64
}

// CreateOrderRepositoryInput carries everything the repository needs to persist
// a new order in a single transaction (counter, number, header, items).
type CreateOrderRepositoryInput struct {
	Input           orderdomain.CreateOrderInput
	Shop            Department
	CreatedAt       time.Time
	FulfillmentDate time.Time
	CounterDay      string
}

// UpdateOrderRepositoryInput carries the resolved update plus the precomputed
// history diff so the repository can persist them atomically.
type UpdateOrderRepositoryInput struct {
	Number            string
	Items             []orderdomain.OrderItem
	FromDepartmentID  *int64
	ToDepartmentID    *int64
	CreatedByUsername string
	FulfillmentDate   time.Time
	Comments          orderdomain.OrderComments
	HistoryItems      []orderdomain.OrderHistoryItem
}

type ListOrdersInput struct {
	Limit            int32
	Offset           int32
	FromDepartmentID *int64
	FulfillmentDate  time.Time
}

type ListOrdersResult struct {
	Orders []orderdomain.Order
	Total  int64
	Limit  int32
	Offset int32
}

type UpdateOrderInput struct {
	Number            string
	Items             []orderdomain.OrderItem
	FromDepartmentID  *int64
	ToDepartmentID    *int64
	CreatedByUsername string
	FulfillmentDate   time.Time
	Comments          orderdomain.OrderComments
}

type EnsureDefaultTemplatesResult struct {
	CatalogItems int
}
