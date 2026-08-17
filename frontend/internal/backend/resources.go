package backend

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

func (c *Client) Login(ctx context.Context, username, password, initData string) (contract.LoginResponse, error) {
	var out contract.LoginResponse
	err := c.do(ctx, "", request{
		method: http.MethodPost,
		path:   "/login",
		body:   contract.LoginRequest{Username: username, Password: password, InitData: initData},
		out:    &out,
	})
	return out, err
}

func (c *Client) Me(ctx context.Context, cred application.Credentials) (contract.Me, error) {
	var out contract.Me
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/me", out: &out})
	return out, err
}

func (c *Client) Departments(ctx context.Context, cred application.Credentials, departmentType string) ([]contract.Department, error) {
	query := url.Values{}
	if departmentType != "" {
		query.Set("type", departmentType)
	}
	var out []contract.Department
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/departments", query: query, out: &out})
	return out, err
}

func (c *Client) Catalog(ctx context.Context, cred application.Credentials) ([]contract.Dish, error) {
	var out []contract.Dish
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/catalog", out: &out})
	return out, err
}

func (c *Client) Categories(ctx context.Context, cred application.Credentials) ([]contract.Category, error) {
	var out []contract.Category
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/categories", out: &out})
	return out, err
}

func (c *Client) Orders(ctx context.Context, cred application.Credentials, filters application.OrderFilters) (contract.OrdersPage, error) {
	query := url.Values{}
	if filters.Page > 0 {
		query.Set("page", strconv.Itoa(filters.Page))
	}
	if filters.Limit > 0 {
		query.Set("limit", strconv.Itoa(filters.Limit))
	}
	setInt64(query, "from_department_id", filters.FromDepartmentID)
	setInt64(query, "category_id", filters.CategoryID)
	setString(query, "fulfillment_date", filters.FulfillmentDate)
	setString(query, "fulfillment_from", filters.FulfillmentFrom)
	setString(query, "fulfillment_to", filters.FulfillmentTo)
	var out contract.OrdersPage
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/orders", query: query, out: &out})
	return out, err
}

func (c *Client) Order(ctx context.Context, cred application.Credentials, number string) (contract.Order, error) {
	var out contract.Order
	err := c.do(ctx, cred, request{method: http.MethodGet, path: resourcePath("/orders/", number), out: &out})
	return out, err
}

func (c *Client) CreateOrder(ctx context.Context, cred application.Credentials, body contract.OrderWrite) (contract.Order, error) {
	var out contract.Order
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/orders", body: body, out: &out})
	return out, err
}

func (c *Client) UpdateOrder(ctx context.Context, cred application.Credentials, number string, body contract.OrderWrite) (contract.Order, error) {
	var out contract.Order
	err := c.do(ctx, cred, request{method: http.MethodPut, path: resourcePath("/orders/", number), body: body, out: &out})
	return out, err
}

func (c *Client) CancelOrder(ctx context.Context, cred application.Credentials, number string) (contract.Order, error) {
	return c.orderAction(ctx, cred, number, "/cancel", http.MethodPost, nil)
}

func (c *Client) RestoreOrder(ctx context.Context, cred application.Credentials, number string) (contract.Order, error) {
	return c.orderAction(ctx, cred, number, "/restore", http.MethodPost, nil)
}

func (c *Client) SetOrderFavorite(ctx context.Context, cred application.Credentials, number string, favorite bool) (contract.Order, error) {
	return c.orderAction(ctx, cred, number, "/favorite", http.MethodPatch, contract.FavoriteUpdate{Favorite: favorite})
}

func (c *Client) SaveOrderDraft(ctx context.Context, cred application.Credentials, body contract.OrderWrite) (contract.OrderDraft, error) {
	var out contract.OrderDraft
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/orders/draft", body: body, out: &out})
	return out, err
}

func (c *Client) OrderDraft(ctx context.Context, cred application.Credentials, categoryID int64) (contract.OrderDraft, error) {
	var out contract.OrderDraft
	err := c.do(ctx, cred, request{method: http.MethodGet, path: idPath("/orders/draft/", categoryID), out: &out})
	return out, err
}

func (c *Client) OrderDrafts(ctx context.Context, cred application.Credentials) ([]contract.OrderDraft, error) {
	var out []contract.OrderDraft
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/orders/drafts", out: &out})
	return out, err
}

func (c *Client) DeleteOrderDraft(ctx context.Context, cred application.Credentials, categoryID int64) error {
	return c.do(ctx, cred, request{method: http.MethodDelete, path: idPath("/orders/draft/", categoryID)})
}

func (c *Client) orderAction(ctx context.Context, cred application.Credentials, number, suffix, method string, body any) (contract.Order, error) {
	var out contract.Order
	err := c.do(ctx, cred, request{method: method, path: resourcePath("/orders/", number) + suffix, body: body, out: &out})
	return out, err
}

func (c *Client) ProductionSheets(ctx context.Context, cred application.Credentials) ([]contract.ProductionSheet, error) {
	var out []contract.ProductionSheet
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/production", out: &out})
	return out, err
}

func (c *Client) ProductionSheet(ctx context.Context, cred application.Credentials, id int64) (contract.ProductionSheet, error) {
	var out contract.ProductionSheet
	err := c.do(ctx, cred, request{method: http.MethodGet, path: idPath("/production/", id), out: &out})
	return out, err
}

func (c *Client) CreateProductionSheet(ctx context.Context, cred application.Credentials, body contract.ProductionWrite) (contract.ProductionSheet, error) {
	var out contract.ProductionSheet
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/production", body: body, out: &out})
	return out, err
}

func (c *Client) UpdateProductionSheet(ctx context.Context, cred application.Credentials, id int64, body contract.ProductionWrite) (contract.ProductionSheet, error) {
	var out contract.ProductionSheet
	err := c.do(ctx, cred, request{method: http.MethodPut, path: idPath("/production/", id), body: body, out: &out})
	return out, err
}

func (c *Client) DeleteProductionSheet(ctx context.Context, cred application.Credentials, id int64) error {
	return c.do(ctx, cred, request{method: http.MethodDelete, path: idPath("/production/", id)})
}

func (c *Client) OrderMonitor(ctx context.Context, cred application.Credentials, number string) (contract.OrderMonitor, error) {
	var out contract.OrderMonitor
	err := c.do(ctx, cred, request{method: http.MethodGet, path: resourcePath("/monitor/", number), out: &out})
	return out, err
}

func (c *Client) BatchMonitor(ctx context.Context, cred application.Credentials, numbers []string) (contract.BatchMonitor, error) {
	query := url.Values{}
	for _, number := range numbers {
		query.Add("orders", number)
	}
	var out contract.BatchMonitor
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/monitor/batch", query: query, out: &out})
	return out, err
}

func (c *Client) CalculateDough(ctx context.Context, cred application.Credentials, body contract.DoughCalcRequest) ([]contract.MonitorReport, error) {
	var out contract.DoughCalcResponse
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/monitor/calc", body: body, out: &out})
	return out.Reports, err
}

func (c *Client) Users(ctx context.Context, cred application.Credentials) ([]contract.User, error) {
	var out []contract.User
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/users", out: &out})
	return out, err
}

func (c *Client) CreateUser(ctx context.Context, cred application.Credentials, body contract.UserCreate) (contract.User, error) {
	var out contract.User
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/users", body: body, out: &out})
	return out, err
}

func (c *Client) UpdateUser(ctx context.Context, cred application.Credentials, id int64, body contract.UserUpdate) (contract.User, error) {
	var out contract.User
	err := c.do(ctx, cred, request{method: http.MethodPatch, path: idPath("/users/", id), body: body, out: &out})
	return out, err
}

func (c *Client) DeleteUser(ctx context.Context, cred application.Credentials, id int64) error {
	return c.do(ctx, cred, request{method: http.MethodDelete, path: idPath("/users/", id)})
}

func (c *Client) AdminDepartments(ctx context.Context, cred application.Credentials) ([]contract.Department, error) {
	var out []contract.Department
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/admin/departments", out: &out})
	return out, err
}

func (c *Client) Dishes(ctx context.Context, cred application.Credentials) ([]contract.Dish, error) {
	var out []contract.Dish
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/admin/dishes", out: &out})
	return out, err
}

func (c *Client) DishTechCards(ctx context.Context, cred application.Credentials) ([]contract.TechCardCategory, error) {
	var out []contract.TechCardCategory
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/admin/dishes/techcards", out: &out})
	return out, err
}

func (c *Client) AvailableDishes(ctx context.Context, cred application.Credentials, queryText string) ([]contract.AvailableDish, error) {
	query := url.Values{}
	setString(query, "q", queryText)
	var out []contract.AvailableDish
	err := c.do(ctx, cred, request{method: http.MethodGet, path: "/admin/dishes/available", query: query, out: &out})
	return out, err
}

func (c *Client) CreateDish(ctx context.Context, cred application.Credentials, body contract.DishWrite) (contract.Dish, error) {
	var out contract.Dish
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/admin/dishes", body: body, out: &out})
	return out, err
}

func (c *Client) UpdateDish(ctx context.Context, cred application.Credentials, code string, body contract.DishWrite) (contract.Dish, error) {
	var out contract.Dish
	err := c.do(ctx, cred, request{method: http.MethodPut, path: resourcePath("/admin/dishes/", code), body: body, out: &out})
	return out, err
}

func (c *Client) DeleteDish(ctx context.Context, cred application.Credentials, code string) error {
	return c.do(ctx, cred, request{method: http.MethodDelete, path: resourcePath("/admin/dishes/", code)})
}

func (c *Client) ReorderDishes(ctx context.Context, cred application.Credentials, codes []string) error {
	return c.do(ctx, cred, request{method: http.MethodPut, path: "/admin/dishes/reorder", body: contract.ReorderDishes{Codes: codes}})
}

func (c *Client) CreateCategory(ctx context.Context, cred application.Credentials, body contract.CategoryWrite) (contract.Category, error) {
	var out contract.Category
	err := c.do(ctx, cred, request{method: http.MethodPost, path: "/admin/categories", body: body, out: &out})
	return out, err
}

func (c *Client) UpdateCategory(ctx context.Context, cred application.Credentials, id int64, body contract.CategoryWrite) (contract.Category, error) {
	var out contract.Category
	err := c.do(ctx, cred, request{method: http.MethodPut, path: idPath("/admin/categories/", id), body: body, out: &out})
	return out, err
}

func (c *Client) DeleteCategory(ctx context.Context, cred application.Credentials, id int64) error {
	return c.do(ctx, cred, request{method: http.MethodDelete, path: idPath("/admin/categories/", id)})
}

func (c *Client) SyncIiko(ctx context.Context, cred application.Credentials) error {
	return c.do(ctx, cred, request{method: http.MethodPost, path: "/admin/iiko/sync"})
}

func resourcePath(prefix, value string) string { return prefix + url.PathEscape(value) }

func idPath(prefix string, id int64) string { return prefix + strconv.FormatInt(id, 10) }

func setInt64(values url.Values, key string, value int64) {
	if value > 0 {
		values.Set(key, strconv.FormatInt(value, 10))
	}
}

func setString(values url.Values, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}
