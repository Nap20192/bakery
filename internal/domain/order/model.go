package order

import (
	"time"

	sharedkernel "bakery/internal/pkg/sharedkernel"
)

// Order — агрегат заказа.
// Содержит шапку заказа и набор позиций, который используется в расчётах и отчётах.
type Order struct {
	sharedkernel.AggregateRoot `json:"-"`

	ID                string      `json:"id"`
	Number            string      `json:"number"`
	Location          string      `json:"location"`
	FromDepartmentID  *int64      `json:"from_department_id"`
	ToDepartmentID    *int64      `json:"to_department_id"`
	CreatedByUsername string      `json:"created_by_username"`
	Items             []OrderItem `json:"items"`
	CreatedAt         time.Time   `json:"created_at"`
	FulfillmentDate   time.Time   `json:"fulfillment_date"`
	History           []OrderHistory
}

type OrderHistory struct {
	ID                int64
	ChangedByUsername string
	ChangedAt         time.Time
	Items             []OrderHistoryItem
}

type OrderHistoryItem struct {
	ChangeType          string
	ProductCode         string
	ProductName         string
	OldQuantity         *float64
	NewQuantity         *float64
	OldReservedQuantity *float64
	NewReservedQuantity *float64
}

type OrderTemplate struct {
	ID              int64
	Name            string
	Body            string
	CreatedByUserID *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DishCatalogItem struct {
	Name      string `json:"name"`
	Theme     string `json:"theme"`
	SortOrder int64  `json:"sort_order"`
}

// CreateOrderInput — входная модель для создания нового заказа.
// Date может быть пустой: тогда сервис подставляет текущее время.
type CreateOrderInput struct {
	Items             []OrderItem
	Location          string
	FromDepartmentID  *int64
	ToDepartmentID    *int64
	CreatedByUsername string
	Date              time.Time
	FulfillmentDate   time.Time
}

// OrderItem — одна позиция в заказе.
// Code — код блюда из iiko, ProductName — отображаемое имя в заявке.
type OrderItem struct {
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
	ProductName      string  `json:"product"`
	Code             string  `json:"code"`
}

func (item OrderItem) ProductionQuantity() float64 {
	return item.Quantity + item.ReservedQuantity
}
