package order

import (
	"time"
)

// Order — агрегат заказа.
// Содержит шапку заказа и набор позиций, который используется в расчётах и отчётах.
type Order struct {
	ID               string      `json:"id"`
	Number           string      `json:"number"`
	Location         string      `json:"location"`
	FromDepartmentID *int64      `json:"from_department_id"`
	ToDepartmentID   *int64      `json:"to_department_id"`
	Items            []OrderItem `json:"items"`
	CreatedAt        time.Time   `json:"created_at"`
}

// CreateOrderInput — входная модель для создания нового заказа.
// Date может быть пустой: тогда сервис подставляет текущее время.
type CreateOrderInput struct {
	Items            []OrderItem
	Location         string
	FromDepartmentID *int64
	ToDepartmentID   *int64
	Date             time.Time
}

// OrderItem — одна позиция в заказе.
// Code — код блюда из iiko, ProductName — отображаемое имя в заявке.
type OrderItem struct {
	Quantity    float64 `json:"quantity"`
	ProductName string  `json:"product"`
	Code        string  `json:"code"`
}
