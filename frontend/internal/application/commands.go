package application

import (
	"context"

	"bakery/internal/inbound/api/contract"
)

type Commands interface {
	Login(ctx context.Context, username, password, initData string) (contract.LoginResponse, error)
	CreateOrder(ctx context.Context, credentials Credentials, input contract.OrderWrite) (contract.Order, error)
	UpdateOrder(ctx context.Context, credentials Credentials, number string, input contract.OrderWrite) (contract.Order, error)
	CancelOrder(ctx context.Context, credentials Credentials, number string) (contract.Order, error)
	RestoreOrder(ctx context.Context, credentials Credentials, number string) (contract.Order, error)
	SetOrderFavorite(ctx context.Context, credentials Credentials, number string, favorite bool) (contract.Order, error)
	SaveOrderDraft(ctx context.Context, credentials Credentials, input contract.OrderWrite) (contract.OrderDraft, error)
	DeleteOrderDraft(ctx context.Context, credentials Credentials, categoryID int64) error
	CreateProductionSheet(ctx context.Context, credentials Credentials, input contract.ProductionWrite) (contract.ProductionSheet, error)
	UpdateProductionSheet(ctx context.Context, credentials Credentials, id int64, input contract.ProductionWrite) (contract.ProductionSheet, error)
	DeleteProductionSheet(ctx context.Context, credentials Credentials, id int64) error
	CalculateDough(ctx context.Context, credentials Credentials, input contract.DoughCalcRequest) ([]contract.MonitorReport, error)
	CreateUser(ctx context.Context, credentials Credentials, input contract.UserCreate) (contract.User, error)
	UpdateUser(ctx context.Context, credentials Credentials, id int64, input contract.UserUpdate) (contract.User, error)
	DeleteUser(ctx context.Context, credentials Credentials, id int64) error
	CreateDish(ctx context.Context, credentials Credentials, input contract.DishWrite) (contract.Dish, error)
	UpdateDish(ctx context.Context, credentials Credentials, code string, input contract.DishWrite) (contract.Dish, error)
	DeleteDish(ctx context.Context, credentials Credentials, code string) error
	ReorderDishes(ctx context.Context, credentials Credentials, codes []string) error
	CreateCategory(ctx context.Context, credentials Credentials, input contract.CategoryWrite) (contract.Category, error)
	UpdateCategory(ctx context.Context, credentials Credentials, id int64, input contract.CategoryWrite) (contract.Category, error)
	DeleteCategory(ctx context.Context, credentials Credentials, id int64) error
	SyncIiko(ctx context.Context, credentials Credentials) error
}
