package order

import (
	"strings"
	"time"

	sharedkernel "bakery/internal/pkg/sharedkernel"
)

// Order — агрегат заказа.
// Содержит шапку заказа и набор позиций, который используется в расчётах и отчётах.
type Order struct {
	sharedkernel.AggregateRoot `json:"-"`

	ID                string        `json:"id"`
	Number            string        `json:"number"`
	Location          string        `json:"location"`
	FromDepartmentID  *int64        `json:"from_department_id"`
	ToDepartmentID    *int64        `json:"to_department_id"`
	CreatedByUsername string        `json:"created_by_username"`
	Items             []OrderItem   `json:"items"`
	CreatedAt         time.Time     `json:"created_at"`
	FulfillmentDate   time.Time     `json:"fulfillment_date"`
	Comments          OrderComments `json:"comments"`
	Favorite          bool          `json:"favorite"`
	// Cancelled marks a soft-cancelled order. Cancelled orders cannot be edited
	// until restored.
	Cancelled           bool   `json:"cancelled"`
	CancelledByUsername string `json:"cancelled_by_username,omitempty"`
	History             []OrderHistory
}

// OrderComments holds an order's free-form notes: per-item comments (keyed by
// product name) and one general comment for the whole order.
type OrderComments struct {
	General string        `json:"general,omitempty"`
	Items   []ItemComment `json:"items,omitempty"`
}

type ItemComment struct {
	ProductName string `json:"product_name"`
	Comment     string `json:"comment"`
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
	Code      string `json:"code"`
	Name      string `json:"name"`
	Theme     string `json:"theme"`
	SortOrder int64  `json:"sort_order"`
}

// AvailableDish is an iiko DISH product the admin can add to the catalog.
type AvailableDish struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Unit string `json:"unit"`
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
	Comments          OrderComments
}

// OrderItem — одна позиция в заказе.
// Code — код блюда из iiko, ProductName — отображаемое имя в заявке.
type OrderItem struct {
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
	ProductName      string  `json:"product"`
	Code             string  `json:"code"`
	// Comment is a transient per-line note captured from bulk input. It is not
	// stored on the item row; the service folds it into the order's comments.
	Comment string `json:"comment,omitempty"`
}

// CommentsFromItems collects per-item comments into OrderComments, keyed by
// product name. Items without a comment are skipped.
func CommentsFromItems(items []OrderItem) OrderComments {
	var comments OrderComments
	for _, item := range items {
		text := strings.TrimSpace(item.Comment)
		if text == "" {
			continue
		}
		comments.Items = append(comments.Items, ItemComment{ProductName: item.ProductName, Comment: text})
	}
	return comments
}

func (item OrderItem) ProductionQuantity() float64 {
	return item.Quantity + item.ReservedQuantity
}
