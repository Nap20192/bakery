package monitoring

// DefaultMonitorCodes — коды теста по умолчанию для расчёта (бот и API).
var DefaultMonitorCodes = []string{"17642", "17644", "17650", "19694"}

// IngredientUsage описывает расход конкретного ингредиента.
// Quantity хранится в Unit (например, кг/шт/л).
type IngredientUsage struct {
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
}

// IngredientComponent — компонент техкарты ингредиента (например, мука в
// тесте), отмасштабированный на общий расход по заказу.
type IngredientComponent struct {
	ProductCode string  `json:"product_code"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
}

// IngredientReport — итог мониторинга по ингредиенту:
// общий расход + детализация по блюдам + состав по техкарте.
type IngredientReport struct {
	Ingredient IngredientUsage           `json:"ingredient"`
	Breakdown  []IngredientDishBreakdown `json:"breakdown"`
	Components []IngredientComponent     `json:"components,omitempty"`
}

// OrderMonitoringReport хранит расчёт по одному заказу внутри batch-расчёта.
type OrderMonitoringReport struct {
	OrderNumber string             `json:"order_number"`
	Reports     []IngredientReport `json:"reports"`
}

// BatchMonitoringReport хранит расчёты по каждому заказу отдельно
// и общий расход ингредиентов по всем заказам.
type BatchMonitoringReport struct {
	Orders       []OrderMonitoringReport `json:"orders"`
	TotalReports []IngredientReport      `json:"total_reports"`
}
