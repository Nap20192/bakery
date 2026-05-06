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
	Quantity      int     `json:"quantity"`
	Product       string  `json:"product"`
	Code          string  `json:"code"`
	IikoProductID *string `json:"iiko_product_id,omitempty"`
}

func CountIngredient(items []Ingredient, targetName string) float64 {
	var total float64
	for _, item := range items {
		if item.Name() == targetName {
			total += item.Quantity()
		}
		subIngredients := item.Ingredients()
		if len(subIngredients) > 0 {
			total += CountIngredient(subIngredients, targetName)
		}
	}
	return total
}

func FindIngredientsRecursive(items []Ingredient, target string) (float64, []Ingredient) {
	var total float64
	var found []Ingredient

	for _, item := range items {
		if item.Name() == target {
			total += item.Quantity()
			found = append(found, item)
		}
		subCount, subFound := FindIngredientsRecursive(item.Ingredients(), target)
		total += subCount
		found = append(found, subFound...)
	}
	return total, found
}
