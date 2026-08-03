package order

import (
	"time"

	sharedkernel "bakery/internal/pkg/sharedkernel"
)

// OrderCategory — тип заявки («Хлеб», «Булочки», …). Динамический справочник:
// админ создаёт новые типы. Letter участвует в номере заказа, Color — слаг
// цвета из фиксированной палитры для выделения заявок на фронте.
// MonitorCodes — коды теста (iiko), по которым считается мониторинг для
// заявок этого типа.
type OrderCategory struct {
	ID           int64    `json:"id"`
	Code         string   `json:"code"`
	Letter       string   `json:"letter"`
	Name         string   `json:"name"`
	Color        string   `json:"color"`
	SortOrder    int64    `json:"sort_order"`
	MonitorCodes []string `json:"monitor_codes"`
}

// Order — агрегат заказа.
// Содержит шапку заказа и набор позиций, который используется в расчётах и отчётах.
type Order struct {
	sharedkernel.AggregateRoot `json:"-"`

	ID                string         `json:"id"`
	Number            string         `json:"number"`
	Location          string         `json:"location"`
	FromDepartmentID  *int64         `json:"from_department_id"`
	ToDepartmentID    *int64         `json:"to_department_id"`
	CategoryID        *int64         `json:"category_id"`
	Category          *OrderCategory `json:"category,omitempty"`
	CreatedByUsername string         `json:"created_by_username"`
	Items             []OrderItem    `json:"items"`
	CreatedAt         time.Time      `json:"created_at"`
	FulfillmentDate   time.Time      `json:"fulfillment_date"`
	Comments          OrderComments  `json:"comments"`
	Favorite          bool           `json:"favorite"`
	// Cancelled marks a soft-cancelled order. Cancelled orders cannot be edited
	// until restored.
	Cancelled           bool   `json:"cancelled"`
	CancelledByUsername string `json:"cancelled_by_username,omitempty"`
	// ProductionSheetID — документ отработки, покрывающий заказ (nil — отработки
	// нет). Заказ принадлежит максимум одному листу; лист покрывает много
	// заказов (1:N).
	ProductionSheetID *int64 `json:"production_sheet_id,omitempty"`
	History           []OrderHistory
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

// ProductionSheet — документ отработки в журнале: хранит закладку каждой
// позиции партии и отклонения фактического выхода. Заказ не изменяется:
// выход декорирует его позиции при чтении (свежий лист побеждает).
type ProductionSheet struct {
	ID                int64                 `json:"id"`
	CreatedByUsername string                `json:"created_by_username"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	Items             []ProductionSheetItem `json:"items,omitempty"`
	// OrderNumbers/ItemCount — сводка для списка журнала.
	OrderNumbers []string `json:"order_numbers"`
	ItemCount    int64    `json:"item_count"`
}

type ProductionSheetItem struct {
	OrderNumber      string  `json:"order_number"`
	ProductName      string  `json:"product_name"`
	LoadedQuantity   float64 `json:"loaded_quantity"`
	ProducedQuantity float64 `json:"produced_quantity"`
	// Reason — необязательное обоснование отклонения.
	Reason string `json:"reason"`
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
	Code       string `json:"code"`
	Name       string `json:"name"`
	Theme      string `json:"theme"`
	CategoryID *int64 `json:"category_id"`
	SortOrder  int64  `json:"sort_order"`
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
	CategoryID        int64
	CreatedByUsername string
	Date              time.Time
	FulfillmentDate   time.Time
	Comments          OrderComments
}

// OrderDraft — черновик заявки: один на пользователя на тип заявки (категорию),
// повторное сохранение перезаписывает предыдущий. Не участвует в нумерации
// заказов и не имеет своих доменных событий — это временная заготовка,
// удаляемая при создании реального заказа из неё же или сборщиком мусора по
// возрасту (см. RunCleanupTicker).
type OrderDraft struct {
	CreatedByUsername string
	CategoryID        int64
	FromDepartmentID  *int64
	Items             []OrderItem
	FulfillmentDate   time.Time
	Comments          OrderComments
	UpdatedAt         time.Time
}

// CategoryColors — допустимые слаги цвета типа заявки. Фронтенд сопоставляет
// слаг с палитрой; хранение произвольного HEX не поддерживается сознательно,
// чтобы карточки оставались в едином стиле.
var CategoryColors = []string{"amber", "sky", "violet", "emerald", "rose", "stone"}

// IsValidCategoryColor reports whether the slug belongs to the fixed palette.
func IsValidCategoryColor(color string) bool {
	for _, c := range CategoryColors {
		if c == color {
			return true
		}
	}
	return false
}

// OrderItem — одна позиция в заказе.
// Code — код блюда из iiko, ProductName — отображаемое имя в заявке.
type OrderItem struct {
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
	// ProducedQuantity — отработка: сколько реально испечено. nil — факт ещё
	// не вносился. Заявка при отработке не изменяется.
	ProducedQuantity *float64 `json:"produced_quantity,omitempty"`
	// ProducedReason — обоснование отклонения («подгорело», «упало», …),
	// проекция причины из документа отработки.
	ProducedReason string `json:"produced_reason,omitempty"`
	ProductName    string `json:"product"`
	Code           string `json:"code"`
	// Comment is a transient per-line note captured from bulk input. It is not
	// stored on the item row.
	Comment string `json:"comment,omitempty"`
}

func (item OrderItem) ProductionQuantity() float64 {
	return item.Quantity + item.ReservedQuantity
}

// EffectiveQuantity — количество для расчётов (мониторинг теста): факт
// отработки, если он внесён, иначе заявка. Отработка хранит только
// отклонения, поэтому nil означает «испечено по заявке».
func (item OrderItem) EffectiveQuantity() float64 {
	if item.ProducedQuantity != nil {
		return *item.ProducedQuantity
	}
	return item.ProductionQuantity()
}
