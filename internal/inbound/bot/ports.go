package bot

import (
	"context"
	"time"

	"bakery/internal/pkg/enum"
	accessdomain "bakery/internal/services/auth/domain"
	departmentuc "bakery/internal/services/department/usecase/department"
	monitoringdomain "bakery/internal/services/monitor/domain"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
	techcarddomain "bakery/internal/services/techcard/domain"
)

// The bot depends on these narrow, bot-owned backend ports rather than on the
// services' full use-case interfaces (interface segregation + dependency
// inversion). Today they are satisfied by the in-process use cases; a future
// HTTP API client only has to implement these methods to decouple the bot from
// the database.

// OrderBackend is the slice of the order service the bot uses.
type OrderBackend interface {
	CreateOrder(ctx context.Context, input orderdomain.CreateOrderInput) (orderdomain.Order, error)
	UpdateOrder(ctx context.Context, input orderuc.UpdateOrderInput) (orderdomain.Order, error)
	GetOrderByNumber(ctx context.Context, number string) (orderdomain.Order, error)
	ListOrders(ctx context.Context, input orderuc.ListOrdersInput) (orderuc.ListOrdersResult, error)
	ValidateBulkOrder(ctx context.Context, order string) orderdomain.BulkOrderValidationResult
	ListOrderTemplates(ctx context.Context) ([]orderdomain.OrderTemplate, error)
	CombinedOrderTemplate(ctx context.Context) (string, error)
	GetOrderTemplate(ctx context.Context, theme string) (orderdomain.OrderTemplate, error)
	GetTemplate(ctx context.Context) (string, error)
}

// AuthBackend is the slice of the auth service the bot uses.
type AuthBackend interface {
	AuthenticateTelegram(ctx context.Context, telegramID int64, telegramUsername, password string) (accessdomain.AuthUser, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (accessdomain.AuthUser, error)
	GetUserByTelegramUsername(ctx context.Context, telegramUsername string) (accessdomain.AuthUser, error)
	ListUsersByRole(ctx context.Context, role string) ([]accessdomain.AuthUser, error)
}

// Authorizer answers RBAC permission checks.
type Authorizer interface {
	HasPermission(role string, permission enum.Permission) bool
}

// DepartmentBackend is the slice of the department service the bot uses.
type DepartmentBackend interface {
	ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]departmentuc.Department, error)
	GetByCode(ctx context.Context, code string) (departmentuc.Department, error)
	GetByID(ctx context.Context, id int64) (departmentuc.Department, error)
}

// MonitorBackend is the slice of the monitoring service the bot uses.
type MonitorBackend interface {
	GetIngredientsByCode(ctx context.Context, code string, order orderdomain.Order) (monitoringdomain.IngredientReport, error)
	GetBatchIngredientsByCodes(ctx context.Context, codes []string, orders []orderdomain.Order) (monitoringdomain.BatchMonitoringReport, error)
}

// SyncBackend is the slice of the iiko sync service the bot uses.
type SyncBackend interface {
	SyncOnce(ctx context.Context) error
}

// TechCardBackend is the slice of the techcard service the bot uses.
type TechCardBackend interface {
	GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error)
}
