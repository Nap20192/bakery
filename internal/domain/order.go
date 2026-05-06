package domain

import (
	"time"
)

type Order struct {
	ID        string      `json:"id"`
	Number    string      `json:"number"`
	Location  string      `json:"location"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
}

type CreateOrderInput struct {
	Items    []OrderItem
	Location string
	Date     time.Time // если zero — используется time.Now()
}

type OrderItem struct {
	Quantity      float64 `json:"quantity"`
	Product       string  `json:"product"`
	Code          string  `json:"code"`
	IikoProductID *string `json:"iiko_product_id,omitempty"`
}
