package application

import (
	"context"

	"bakery/internal/inbound/api/contract"
)

type Queries interface {
	Health(ctx context.Context) error
	Me(ctx context.Context, credentials Credentials) (contract.Me, error)
	Departments(ctx context.Context, credentials Credentials, departmentType string) ([]contract.Department, error)
	Catalog(ctx context.Context, credentials Credentials) ([]contract.Dish, error)
	Categories(ctx context.Context, credentials Credentials) ([]contract.Category, error)
	Orders(ctx context.Context, credentials Credentials, filters OrderFilters) (contract.OrdersPage, error)
	Order(ctx context.Context, credentials Credentials, number string) (contract.Order, error)
	ProductionSheets(ctx context.Context, credentials Credentials) ([]contract.ProductionSheet, error)
	ProductionSheet(ctx context.Context, credentials Credentials, id int64) (contract.ProductionSheet, error)
	OrderMonitor(ctx context.Context, credentials Credentials, number string) (contract.OrderMonitor, error)
	BatchMonitor(ctx context.Context, credentials Credentials, numbers []string) (contract.BatchMonitor, error)
	Users(ctx context.Context, credentials Credentials) ([]contract.User, error)
	AdminDepartments(ctx context.Context, credentials Credentials) ([]contract.Department, error)
	Dishes(ctx context.Context, credentials Credentials) ([]contract.Dish, error)
	AvailableDishes(ctx context.Context, credentials Credentials, query string) ([]contract.AvailableDish, error)
}
