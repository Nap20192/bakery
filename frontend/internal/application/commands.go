package application

import (
	"context"

	"bakery/internal/inbound/api/contract"
)

type CommandGateway interface {
	Login(ctx context.Context, username, password string) (contract.LoginResponse, error)
	CreateOrder(ctx context.Context, credentials Credentials, input contract.OrderWrite) (contract.Order, error)
	UpdateOrder(ctx context.Context, credentials Credentials, number string, input contract.OrderWrite) (contract.Order, error)
	CancelOrder(ctx context.Context, credentials Credentials, number string) (contract.Order, error)
	RestoreOrder(ctx context.Context, credentials Credentials, number string) (contract.Order, error)
	SetOrderFavorite(ctx context.Context, credentials Credentials, number string, favorite bool) (contract.Order, error)
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
}

type Commands struct {
	gateway CommandGateway
}

func NewCommands(gateway CommandGateway) *Commands {
	return &Commands{gateway: gateway}
}

func (c *Commands) Login(ctx context.Context, username, password string) (contract.LoginResponse, error) {
	return c.gateway.Login(ctx, username, password)
}

func (c *Commands) CreateOrder(ctx context.Context, credentials Credentials, input contract.OrderWrite) (contract.Order, error) {
	return c.gateway.CreateOrder(ctx, credentials, input)
}

func (c *Commands) UpdateOrder(ctx context.Context, credentials Credentials, number string, input contract.OrderWrite) (contract.Order, error) {
	return c.gateway.UpdateOrder(ctx, credentials, number, input)
}

func (c *Commands) CancelOrder(ctx context.Context, credentials Credentials, number string) (contract.Order, error) {
	return c.gateway.CancelOrder(ctx, credentials, number)
}

func (c *Commands) RestoreOrder(ctx context.Context, credentials Credentials, number string) (contract.Order, error) {
	return c.gateway.RestoreOrder(ctx, credentials, number)
}

func (c *Commands) SetOrderFavorite(ctx context.Context, credentials Credentials, number string, favorite bool) (contract.Order, error) {
	return c.gateway.SetOrderFavorite(ctx, credentials, number, favorite)
}

func (c *Commands) CreateProductionSheet(ctx context.Context, credentials Credentials, input contract.ProductionWrite) (contract.ProductionSheet, error) {
	return c.gateway.CreateProductionSheet(ctx, credentials, input)
}

func (c *Commands) UpdateProductionSheet(ctx context.Context, credentials Credentials, id int64, input contract.ProductionWrite) (contract.ProductionSheet, error) {
	return c.gateway.UpdateProductionSheet(ctx, credentials, id, input)
}

func (c *Commands) DeleteProductionSheet(ctx context.Context, credentials Credentials, id int64) error {
	return c.gateway.DeleteProductionSheet(ctx, credentials, id)
}

func (c *Commands) CalculateDough(ctx context.Context, credentials Credentials, input contract.DoughCalcRequest) ([]contract.MonitorReport, error) {
	return c.gateway.CalculateDough(ctx, credentials, input)
}

func (c *Commands) CreateUser(ctx context.Context, credentials Credentials, input contract.UserCreate) (contract.User, error) {
	return c.gateway.CreateUser(ctx, credentials, input)
}

func (c *Commands) UpdateUser(ctx context.Context, credentials Credentials, id int64, input contract.UserUpdate) (contract.User, error) {
	return c.gateway.UpdateUser(ctx, credentials, id, input)
}

func (c *Commands) DeleteUser(ctx context.Context, credentials Credentials, id int64) error {
	return c.gateway.DeleteUser(ctx, credentials, id)
}

func (c *Commands) CreateDish(ctx context.Context, credentials Credentials, input contract.DishWrite) (contract.Dish, error) {
	return c.gateway.CreateDish(ctx, credentials, input)
}

func (c *Commands) UpdateDish(ctx context.Context, credentials Credentials, code string, input contract.DishWrite) (contract.Dish, error) {
	return c.gateway.UpdateDish(ctx, credentials, code, input)
}

func (c *Commands) DeleteDish(ctx context.Context, credentials Credentials, code string) error {
	return c.gateway.DeleteDish(ctx, credentials, code)
}

func (c *Commands) ReorderDishes(ctx context.Context, credentials Credentials, codes []string) error {
	return c.gateway.ReorderDishes(ctx, credentials, codes)
}

func (c *Commands) CreateCategory(ctx context.Context, credentials Credentials, input contract.CategoryWrite) (contract.Category, error) {
	return c.gateway.CreateCategory(ctx, credentials, input)
}

func (c *Commands) UpdateCategory(ctx context.Context, credentials Credentials, id int64, input contract.CategoryWrite) (contract.Category, error) {
	return c.gateway.UpdateCategory(ctx, credentials, id, input)
}

func (c *Commands) DeleteCategory(ctx context.Context, credentials Credentials, id int64) error {
	return c.gateway.DeleteCategory(ctx, credentials, id)
}
