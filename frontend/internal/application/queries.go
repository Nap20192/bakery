package application

import (
	"context"

	"bakery/internal/inbound/api/contract"
)

type QueryGateway interface {
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

type Queries struct {
	gateway QueryGateway
}

func NewQueries(gateway QueryGateway) *Queries {
	return &Queries{gateway: gateway}
}

func (q *Queries) Health(ctx context.Context) error {
	return q.gateway.Health(ctx)
}

func (q *Queries) Me(ctx context.Context, credentials Credentials) (contract.Me, error) {
	return q.gateway.Me(ctx, credentials)
}

func (q *Queries) Departments(ctx context.Context, credentials Credentials, departmentType string) ([]contract.Department, error) {
	return q.gateway.Departments(ctx, credentials, departmentType)
}

func (q *Queries) Catalog(ctx context.Context, credentials Credentials) ([]contract.Dish, error) {
	return q.gateway.Catalog(ctx, credentials)
}

func (q *Queries) Categories(ctx context.Context, credentials Credentials) ([]contract.Category, error) {
	return q.gateway.Categories(ctx, credentials)
}

func (q *Queries) Orders(ctx context.Context, credentials Credentials, filters OrderFilters) (contract.OrdersPage, error) {
	return q.gateway.Orders(ctx, credentials, filters)
}

func (q *Queries) Order(ctx context.Context, credentials Credentials, number string) (contract.Order, error) {
	return q.gateway.Order(ctx, credentials, number)
}

func (q *Queries) ProductionSheets(ctx context.Context, credentials Credentials) ([]contract.ProductionSheet, error) {
	return q.gateway.ProductionSheets(ctx, credentials)
}

func (q *Queries) ProductionSheet(ctx context.Context, credentials Credentials, id int64) (contract.ProductionSheet, error) {
	return q.gateway.ProductionSheet(ctx, credentials, id)
}

func (q *Queries) OrderMonitor(ctx context.Context, credentials Credentials, number string) (contract.OrderMonitor, error) {
	return q.gateway.OrderMonitor(ctx, credentials, number)
}

func (q *Queries) BatchMonitor(ctx context.Context, credentials Credentials, numbers []string) (contract.BatchMonitor, error) {
	return q.gateway.BatchMonitor(ctx, credentials, numbers)
}

func (q *Queries) Users(ctx context.Context, credentials Credentials) ([]contract.User, error) {
	return q.gateway.Users(ctx, credentials)
}

func (q *Queries) AdminDepartments(ctx context.Context, credentials Credentials) ([]contract.Department, error) {
	return q.gateway.AdminDepartments(ctx, credentials)
}

func (q *Queries) Dishes(ctx context.Context, credentials Credentials) ([]contract.Dish, error) {
	return q.gateway.Dishes(ctx, credentials)
}

func (q *Queries) AvailableDishes(ctx context.Context, credentials Credentials, query string) ([]contract.AvailableDish, error) {
	return q.gateway.AvailableDishes(ctx, credentials, query)
}
