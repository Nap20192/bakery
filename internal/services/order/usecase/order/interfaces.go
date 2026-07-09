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
	CancelOrder(ctx context.Context, number, byUsername string) (orderdomain.Order, error)
	RestoreOrder(ctx context.Context, number, byUsername string) (orderdomain.Order, error)
	CreateProductionSheet(ctx context.Context, input RecordProductionInput) (orderdomain.ProductionSheet, error)
	ListProductionSheets(ctx context.Context) ([]orderdomain.ProductionSheet, error)
	GetProductionSheet(ctx context.Context, id int64) (orderdomain.ProductionSheet, error)
	UpdateProductionSheet(ctx context.Context, id int64, input RecordProductionInput) (orderdomain.ProductionSheet, error)
	DeleteProductionSheet(ctx context.Context, id int64, byUsername string) error
	ValidateBulkOrder(ctx context.Context, order string) orderdomain.BulkOrderValidationResult
	ListOrderCategories(ctx context.Context) ([]orderdomain.OrderCategory, error)
	CreateOrderCategory(ctx context.Context, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error)
	UpdateOrderCategory(ctx context.Context, id int64, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error)
	DeleteOrderCategory(ctx context.Context, id int64) error
	ListDishCatalog(ctx context.Context) ([]orderdomain.DishCatalogItem, error)
	SearchAvailableDishes(ctx context.Context, query string, limit int) ([]orderdomain.AvailableDish, error)
	AddDishCatalogItem(ctx context.Context, input orderdomain.DishCatalogItem) (orderdomain.DishCatalogItem, error)
	UpdateDishCatalogItem(ctx context.Context, code string, input orderdomain.DishCatalogItem) (orderdomain.DishCatalogItem, error)
	ReorderDishCatalog(ctx context.Context, codes []string) error
	DeleteDishCatalogItem(ctx context.Context, code string) error
	ListOrderTemplates(ctx context.Context) ([]orderdomain.OrderTemplate, error)
	CombinedOrderTemplate(ctx context.Context) (string, error)
	GetOrderTemplate(ctx context.Context, theme string) (orderdomain.OrderTemplate, error)
	GetTemplate(ctx context.Context) (string, error)
	EnsureDefaultOrderTemplates(ctx context.Context, seeds ...CatalogSeed) (EnsureDefaultTemplatesResult, error)
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
	CancelOrder(ctx context.Context, number, by string) (orderdomain.Order, error)
	RestoreOrder(ctx context.Context, number string) (orderdomain.Order, error)
	SaveProductionSheet(ctx context.Context, input SaveProductionSheetInput) (orderdomain.ProductionSheet, error)
	ListProductionSheets(ctx context.Context) ([]orderdomain.ProductionSheet, error)
	GetProductionSheet(ctx context.Context, id int64) (orderdomain.ProductionSheet, error)
	DeleteProductionSheet(ctx context.Context, id int64, byUsername string) error
	GetDepartmentByID(ctx context.Context, id int64) (Department, error)
	ListOrderCategories(ctx context.Context) ([]orderdomain.OrderCategory, error)
	GetOrderCategoryByID(ctx context.Context, id int64) (orderdomain.OrderCategory, error)
	CreateOrderCategory(ctx context.Context, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error)
	UpdateOrderCategory(ctx context.Context, id int64, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error)
	DeleteOrderCategory(ctx context.Context, id int64) error
	CountDishesByCategoryID(ctx context.Context, id int64) (int64, error)
	DishExistsByCode(ctx context.Context, code string) (bool, error)
	ResolveDishCatalogItem(ctx context.Context, name string) (DishCatalogItem, error)
	ListDishCatalog(ctx context.Context) ([]DishCatalogItem, error)
	SearchAvailableDishes(ctx context.Context, query string, limit int) ([]orderdomain.AvailableDish, error)
	UpsertDishCatalogItem(ctx context.Context, item DishCatalogItem) error
	UpdateDishCatalogItem(ctx context.Context, code string, item DishCatalogItem) (DishCatalogItem, error)
	SetDishCatalogSortOrder(ctx context.Context, code string, sortOrder int64) error
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
	Code       string
	Name       string
	Theme      string
	CategoryID *int64
	SortOrder  int64
}

// CreateOrderRepositoryInput carries everything the repository needs to persist
// a new order in a single transaction (counter, number, header, items).
type CreateOrderRepositoryInput struct {
	Input           orderdomain.CreateOrderInput
	Shop            Department
	Category        orderdomain.OrderCategory
	CreatedAt       time.Time
	FulfillmentDate time.Time
	CounterDay      string
}

// UpdateOrderRepositoryInput carries the resolved update plus the precomputed
// history diff so the repository can persist them atomically.
type UpdateOrderRepositoryInput struct {
	Number string
	Items  []orderdomain.OrderItem
	// ChangedByUsername is the editor who made this change. It is recorded only
	// in order_history — the order's original author is never overwritten.
	ChangedByUsername string
	FromDepartmentID  *int64
	ToDepartmentID    *int64
	FulfillmentDate   time.Time
	Comments          orderdomain.OrderComments
	HistoryItems      []orderdomain.OrderHistoryItem
}

type ListOrdersInput struct {
	Limit            int32
	Offset           int32
	FromDepartmentID *int64
	CategoryID       *int64
	FulfillmentDate  time.Time
	// FulfillmentFrom/To ограничивают выборку диапазоном дат выполнения
	// (включительно) — матрица пекаря грузит заказы окнами по несколько дней.
	FulfillmentFrom time.Time
	FulfillmentTo   time.Time
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

// CatalogSeed — файл-шаблон каталога и тип заявки, к которому относятся его
// блюда. Без аргументов сервис сидит дефолтные пары: dishes.txt → buns,
// bread.txt → bread.
type CatalogSeed struct {
	Path         string
	CategoryCode string
}

// RecordProductionInput — отработка (факт выпечки) по нескольким заказам
// одной партией: пекарь вводит факт на листе выпечки и разносит по заявкам.
type RecordProductionInput struct {
	ProducedByUsername string
	Orders             []OrderProductionInput
}

type OrderProductionInput struct {
	Number string
	Items  []ProducedItemInput
}

type ProducedItemInput struct {
	ProductName      string
	ProducedQuantity float64
	// Reason — необязательное обоснование отклонения («подгорело», «упало»).
	Reason string
}

// SaveProductionSheetInput persists a production-sheet document. SheetID == 0
// creates a new sheet; otherwise the sheet's items are replaced. The
// repository re-projects produced_quantity on every affected order, writes
// history diffs and emits domain events atomically.
type SaveProductionSheetInput struct {
	SheetID            int64
	ProducedByUsername string
	Orders             []OrderProductionInput
}
