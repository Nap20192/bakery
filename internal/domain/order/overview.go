package order

import (
	"time"

	"github.com/google/uuid"
)

type IngredientLine struct {
	IngredientID   uuid.UUID
	IngredientName string
	Unit           string
	TotalQty       float64
}

type TechCardNode struct {
	ProductID   uuid.UUID
	ProductName string
	Unit        string
	Qty         float64
	Ingredients []IngredientLine
	SubProducts []*TechCardNode
}

type OrderItemOverview struct {
	ProductID       uuid.UUID
	ProductName     string
	Quantity        float64
	Unit            string
	Tree            *TechCardNode
	FlatIngredients []IngredientLine
}

type OrderOverview struct {
	OrderID          uuid.UUID
	OrderDate        time.Time
	Status           Status
	Client           string
	Kitchen          string
	Note             string
	Items            []OrderItemOverview
	TotalIngredients []IngredientLine
}

type DayGroup struct {
	Date             time.Time
	Orders           []OrderOverview
	TotalIngredients []IngredientLine
}

type PeriodOverview struct {
	From       time.Time
	To         time.Time
	Days       []DayGroup
	GrandTotal []IngredientLine
}
