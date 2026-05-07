package domain

// IngredientUsage описывает расход конкретного ингредиента.
// Quantity хранится в Unit (например, кг/шт/л).
type IngredientUsage struct {
	ProductID   string  `json:"product_id"`
	ProductCode string  `json:"product_code"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
}

// IngredientDishBreakdown — вклад одного блюда заказа
// в общий расход выбранного ингредиента.
type IngredientDishBreakdown struct {
	OrderItemCode      string  `json:"order_item_code"`
	OrderItemName      string  `json:"order_item_name"`
	OrderItemQuantity  float64 `json:"order_item_quantity"`
	IngredientQuantity float64 `json:"ingredient_quantity"`
	ProportionOfTotal  float64 `json:"proportion_of_total"`
}

// IngredientReport — итог мониторинга по ингредиенту:
// общий расход + детализация по блюдам + предупреждения.
type IngredientReport struct {
	Ingredient IngredientUsage           `json:"ingredient"`
	Breakdown  []IngredientDishBreakdown `json:"breakdown"`
	Warnings   []string                  `json:"warnings"`
}
