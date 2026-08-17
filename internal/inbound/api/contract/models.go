// Package contract defines the JSON request and response models shared by the
// worker HTTP delivery adapters and internal HTTP clients.
package contract

import monitoringdomain "bakery/internal/services/monitor/domain"

type Error struct {
	Error string `json:"error"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// InitData is the raw Telegram initData, sent by the Mini App password
	// fallback so the backend can bind telegram_id after a successful login.
	InitData string `json:"init_data,omitempty"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type Me struct {
	Role             string `json:"role"`
	TelegramID       int64  `json:"telegram_id"`
	TelegramUsername string `json:"telegram_username"`
	DepartmentID     int64  `json:"department_id"`
	DepartmentCode   string `json:"department_code"`
	DepartmentName   string `json:"department_name"`
	DepartmentType   string `json:"department_type"`
}

type Department struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type User struct {
	ID               int64  `json:"id"`
	Username         string `json:"username"`
	TelegramUsername string `json:"telegram_username"`
	TelegramID       *int64 `json:"telegram_id"`
	Role             string `json:"role"`
	DepartmentID     *int64 `json:"department_id"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type UserCreate struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	TelegramUsername string `json:"telegram_username,omitempty"`
	Role             string `json:"role"`
	DepartmentCode   string `json:"department_code,omitempty"`
}

type UserUpdate struct {
	Username       *string `json:"username,omitempty"`
	Password       *string `json:"password,omitempty"`
	Role           *string `json:"role,omitempty"`
	DepartmentCode *string `json:"department_code,omitempty"`
}

type Category struct {
	ID           int64    `json:"id"`
	Code         string   `json:"code"`
	Letter       string   `json:"letter"`
	Name         string   `json:"name"`
	Color        string   `json:"color"`
	SortOrder    int64    `json:"sort_order"`
	MonitorCodes []string `json:"monitor_codes"`
}

type CategoryWrite struct {
	Code         string   `json:"code"`
	Letter       string   `json:"letter"`
	Name         string   `json:"name"`
	Color        string   `json:"color"`
	SortOrder    int64    `json:"sort_order,omitempty"`
	MonitorCodes []string `json:"monitor_codes,omitempty"`
}

type OrderItem struct {
	Code               string   `json:"code"`
	ProductName        string   `json:"product_name"`
	Quantity           float64  `json:"quantity"`
	ReservedQuantity   float64  `json:"reserved_quantity"`
	ProductionQuantity float64  `json:"production_quantity"`
	ProducedQuantity   *float64 `json:"produced_quantity,omitempty"`
	ProducedReason     string   `json:"produced_reason,omitempty"`
}

type ItemComment struct {
	ProductName string `json:"product_name"`
	Comment     string `json:"comment"`
}

type Comments struct {
	General string        `json:"general"`
	Items   []ItemComment `json:"items"`
}

type HistoryItem struct {
	ChangeType          string   `json:"change_type"`
	ProductCode         string   `json:"product_code"`
	ProductName         string   `json:"product_name"`
	OldQuantity         *float64 `json:"old_quantity,omitempty"`
	NewQuantity         *float64 `json:"new_quantity,omitempty"`
	OldReservedQuantity *float64 `json:"old_reserved_quantity,omitempty"`
	NewReservedQuantity *float64 `json:"new_reserved_quantity,omitempty"`
}

type HistoryEntry struct {
	ID                int64         `json:"id"`
	ChangedByUsername string        `json:"changed_by_username"`
	ChangedAt         string        `json:"changed_at"`
	Items             []HistoryItem `json:"items"`
}

type Order struct {
	ID                  string         `json:"id"`
	Number              string         `json:"number"`
	Location            string         `json:"location"`
	CreatedByUsername   string         `json:"created_by_username"`
	FromDepartment      *Department    `json:"from_department,omitempty"`
	ToDepartment        *Department    `json:"to_department,omitempty"`
	Category            *Category      `json:"category,omitempty"`
	Items               []OrderItem    `json:"items"`
	CreatedAt           string         `json:"created_at"`
	FulfillmentDate     string         `json:"fulfillment_date"`
	MonitorCommand      string         `json:"monitor_command"`
	Comments            Comments       `json:"comments"`
	Favorite            bool           `json:"favorite"`
	Cancelled           bool           `json:"cancelled"`
	CancelledByUsername string         `json:"cancelled_by_username,omitempty"`
	ProductionSheetID   *int64         `json:"production_sheet_id,omitempty"`
	History             []HistoryEntry `json:"history,omitempty"`
}

type OrdersPage struct {
	Items      []Order `json:"items"`
	Page       int32   `json:"page"`
	Limit      int32   `json:"limit"`
	Offset     int32   `json:"offset"`
	Total      int64   `json:"total"`
	TotalPages int32   `json:"total_pages"`
}

type OrderItemWrite struct {
	ProductName      string  `json:"product_name"`
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
}

type OrderWrite struct {
	Items            []OrderItemWrite `json:"items"`
	FulfillmentDate  string           `json:"fulfillment_date"`
	FromDepartmentID *int64           `json:"from_department_id"`
	CategoryID       int64            `json:"category_id"`
	Comments         Comments         `json:"comments"`
}

// OrderDraft is a saved, unfinished OrderWrite — one per user per category.
type OrderDraft struct {
	Write     OrderWrite `json:"write"`
	UpdatedAt string     `json:"updated_at"`
}

type FavoriteUpdate struct {
	Favorite bool `json:"favorite"`
}

type Dish struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Theme      string `json:"theme"`
	CategoryID *int64 `json:"category_id"`
	SortOrder  int64  `json:"sort_order"`
}

type DishWrite struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Theme      string `json:"theme"`
	CategoryID *int64 `json:"category_id"`
	SortOrder  int64  `json:"sort_order"`
}

type TechCardIngredient struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Unit   string  `json:"unit"`
	Amount float64 `json:"amount"`
}

type TechCardDish struct {
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Unit        string               `json:"unit"`
	Ingredients []TechCardIngredient `json:"ingredients"`
	Error       string               `json:"error,omitempty"`
}

type TechCardCategory struct {
	ID     int64          `json:"id"`
	Name   string         `json:"name"`
	Dishes []TechCardDish `json:"dishes"`
}

type AvailableDish struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

type ReorderDishes struct {
	Codes []string `json:"codes"`
}

type ProductionItemWrite struct {
	ProductName      string   `json:"product_name"`
	LoadedQuantity   *float64 `json:"loaded_quantity"`
	ProducedQuantity float64  `json:"produced_quantity"`
	Reason           string   `json:"reason"`
}

type ProductionOrderWrite struct {
	Number string                `json:"number"`
	Items  []ProductionItemWrite `json:"items"`
}

type ProductionWrite struct {
	Orders []ProductionOrderWrite `json:"orders"`
}

type ProductionSheetItem struct {
	OrderNumber      string  `json:"order_number"`
	ProductName      string  `json:"product_name"`
	LoadedQuantity   float64 `json:"loaded_quantity"`
	ProducedQuantity float64 `json:"produced_quantity"`
	Reason           string  `json:"reason"`
}

type ProductionSheet struct {
	ID                int64                 `json:"id"`
	CreatedByUsername string                `json:"created_by_username"`
	CreatedAt         string                `json:"created_at"`
	UpdatedAt         string                `json:"updated_at"`
	OrderNumbers      []string              `json:"order_numbers"`
	ItemCount         int64                 `json:"item_count"`
	Items             []ProductionSheetItem `json:"items,omitempty"`
}

type DoughCalcItem struct {
	Code        string  `json:"code"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
}

type DoughCalcRequest struct {
	CategoryID int64           `json:"category_id"`
	Items      []DoughCalcItem `json:"items"`
}

type MonitorReport struct {
	Code   string                            `json:"code"`
	Report monitoringdomain.IngredientReport `json:"report"`
}

type OrderMonitor struct {
	Reports []MonitorReport `json:"reports"`
	Order   Order           `json:"order"`
}

type BatchOrderMonitor struct {
	Order   Order           `json:"order"`
	Reports []MonitorReport `json:"reports"`
}

type BatchMonitor struct {
	Orders       []BatchOrderMonitor `json:"orders"`
	TotalReports []MonitorReport     `json:"total_reports"`
}

type DoughCalcResponse struct {
	Reports []MonitorReport `json:"reports"`
}
