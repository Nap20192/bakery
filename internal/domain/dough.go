package domain

import "fmt"

type Dough struct {
	Name        string       `json:"name"`
	Quantity    float64      `json:"quantity"`
	Unit        string       `json:"unit"`
	Ingredients []Ingredient `json:"ingredients,omitempty"`
}

type DoughSummary struct {
	DoughName string
	TotalKg   float64
}

type DoughRepository interface {
	Get(name string) (Dough, error)
	List() ([]Dough, error)
	IsDough(name string) bool
}

type OrderRepository interface {
	Create(items []OrderItem) (Order, error)
	GetByNumber(number string) (Order, error)
	List(limit int) ([]Order, error)
}

var ErrNotFound = fmt.Errorf("не найдено")
