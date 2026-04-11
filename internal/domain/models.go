package domain

import (
	"time"

	"github.com/google/uuid"
)

// Client — заказчик, отправляющий заявки через Telegram.
type Client struct {
	ID             uuid.UUID
	Name           string
	TelegramChatID int64
	Phone          string
	Note           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// Kitchen — производственная площадка (пекарня / цех).
// IikoID привязывает к подразделению (department) в iiko.
type Kitchen struct {
	ID        uuid.UUID
	Name      string
	Address   string
	IikoID    string // iiko department UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Ingredient — сырьё / полуфабрикат.
// IsDough=true означает, что это «тесто» и подлежит расчёту.
type Ingredient struct {
	ID        uuid.UUID
	IikoID    string // product UUID в iiko
	Name      string
	Unit      string // кг, г, л, мл, шт
	IsDough   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Product — готовое изделие (булка, батон, пирожок …).
type Product struct {
	ID        uuid.UUID
	IikoID    string // product UUID в iiko
	Name      string
	Unit      string // шт / кг
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// ─── Технологические карты (ТТК) ────────────────────────────────────────────

// TechCardSource — откуда взята ТТК.
type TechCardSource string

const (
	SourceIiko   TechCardSource = "iiko"
	SourceExcel  TechCardSource = "excel"
	SourceManual TechCardSource = "manual"
)

// TechCard — технологическая карта продукта.
//
// Маппинг из iiko:
//   - iiko.AssemblyChartDto.ID         → SourceRef  (ссылка на карту в iiko)
//   - iiko.AssemblyChartDto.DateFrom   → ValidFrom
//   - iiko.AssemblyChartDto.DateTo     → ValidTo
//   - iiko.AssemblyChartDto.AssembledAmount → YieldQty (выход готового продукта)
//   - iiko.AssemblyChartDto.AssembledProductID → Product.IikoID (через lookup)
type TechCard struct {
	ID        uuid.UUID
	ProductID uuid.UUID // → products.id
	Version   int
	Source    TechCardSource
	SourceRef string  // id карты в iiko / имя файла excel
	YieldQty  float64 // выход готового продукта
	YieldUnit string
	ValidFrom time.Time
	ValidTo   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	Items []TechCardItem // состав
}

// TechCardItem — строка состава ТТК (один ингредиент).
//
// Маппинг из iiko:
//   - iiko.AssemblyChartItem.ProductID → Ingredient.IikoID (через lookup)
//   - iiko.AssemblyChartItem.AmountIn  → GrossQty  (брутто — до обработки)
//   - iiko.AssemblyChartItem.AmountOut → NetQty    (нетто — после обработки)
//
// Для расчёта теста используется PreparedChart (разложенная ТТК):
//   - iiko.PreparedChartItem.ProductID → Ingredient.IikoID
//   - iiko.PreparedChartItem.Amount    → NetQty
type TechCardItem struct {
	ID           uuid.UUID
	TechCardID   uuid.UUID // → tech_cards.id
	IngredientID uuid.UUID // → ingredients.id
	GrossQty     float64   // брутто
	NetQty       float64   // нетто
	Unit         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ─── Заявки ─────────────────────────────────────────────────────────────────

// OrderStatus — статус заявки.
type OrderStatus string

const (
	OrderNew       OrderStatus = "new"       // получена из Telegram
	OrderParsed    OrderStatus = "parsed"    // распарсена в позиции
	OrderConfirmed OrderStatus = "confirmed" // подтверждена оператором
	OrderProcessed OrderStatus = "processed" // тесто рассчитано
	OrderCancelled OrderStatus = "cancelled"
)

// Order — заявка от клиента на дату.
type Order struct {
	ID                uuid.UUID
	ClientID          uuid.UUID
	KitchenID         *uuid.UUID
	TelegramChatID    int64
	TelegramMessageID int64
	OrderDate         time.Time // дата, на которую нужна продукция
	Status            OrderStatus
	RawText           string // исходное сообщение из Telegram
	Note              string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time

	Items []OrderItem
}

// OrderItem — позиция заявки: «100 шт булок», «50 кг батонов».
type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  float64
	Unit      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ─── Расчёт теста ───────────────────────────────────────────────────────────

// DoughCalculation — итоговый расчёт: сколько теста нужно на дату.
//
// Алгоритм:
//  1. Собрать все OrderItem на calc_date
//  2. Для каждого OrderItem найти актуальную TechCard продукта
//  3. В составе TechCard найти ингредиенты с IsDough=true
//  4. dough_per_unit = ingredient.NetQty / tech_card.YieldQty
//  5. dough_qty = order_item.Quantity × dough_per_unit
//  6. Суммировать по типу теста → TotalQty
type DoughCalculation struct {
	ID                uuid.UUID
	KitchenID         *uuid.UUID
	CalcDate          time.Time
	DoughIngredientID uuid.UUID // → ingredients.id (конкретный тип теста)
	TotalQty          float64
	Unit              string
	Source            string // auto / manual
	CreatedAt         time.Time

	Items []DoughCalcItem
}

// DoughCalcItem — детализация: сколько теста ушло на одну позицию заявки.
type DoughCalcItem struct {
	ID            uuid.UUID
	CalculationID uuid.UUID
	OrderItemID   uuid.UUID
	ProductID     uuid.UUID
	TechCardID    uuid.UUID
	Quantity      float64 // количество продукции из заявки
	DoughQty      float64 // сколько теста нужно на эту позицию
	Unit          string
	CreatedAt     time.Time
}
