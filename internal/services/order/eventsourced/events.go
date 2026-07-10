// Package eventsourced — event-sourced ядро заказа (ветка arch/event-sourcing).
//
// Библиотека: github.com/hallgren/eventsourcing. Поток событий заказа —
// источник истины; существующие таблицы orders/order_items остаются read
// model (CQRS) и наполняются проекцией.
//
// События — контракты: структуры ниже сериализуются в event store и не
// должны ломаться задним числом. Меняешь смысл поля — заводи новое событие
// (V2), а не переписывай старое.
package eventsourced

import "time"

// Item — позиция заказа внутри событий (самодостаточная копия, без ссылок на
// domain-пакет, чтобы контракт события не зависел от рефакторингов домена).
type Item struct {
	Code             string  `json:"code"`
	ProductName      string  `json:"product_name"`
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
}

type ItemComment struct {
	ProductName string `json:"product_name"`
	Comment     string `json:"comment"`
}

type Comments struct {
	General string        `json:"general,omitempty"`
	Items   []ItemComment `json:"items,omitempty"`
}

// Created — заказ создан магазином. Номер генерируется на командной стороне
// (счётчик — вне агрегата) и фиксируется в событии.
type Created struct {
	Number            string    `json:"number"`
	Location          string    `json:"location"`
	FromDepartmentID  *int64    `json:"from_department_id"`
	ToDepartmentID    *int64    `json:"to_department_id"`
	CategoryID        *int64    `json:"category_id"`
	CreatedByUsername string    `json:"created_by_username"`
	CreatedAt         time.Time `json:"created_at"`
	FulfillmentDate   time.Time `json:"fulfillment_date"`
	Items             []Item    `json:"items"`
	Comments          Comments  `json:"comments"`
}

// ItemsUpdated — магазин изменил состав/дату/комментарии. Автор заказа не
// меняется, редактор фиксируется в событии.
type ItemsUpdated struct {
	Items             []Item    `json:"items"`
	FulfillmentDate   time.Time `json:"fulfillment_date"`
	Comments          Comments  `json:"comments"`
	ChangedByUsername string    `json:"changed_by_username"`
}

// Cancelled — мягкая отмена заказа.
type Cancelled struct {
	ByUsername string `json:"by_username"`
}

// Restored — отмена снята.
type Restored struct {
	ByUsername string `json:"by_username"`
}

// ProducedItem — отклонение отработки по позиции (только отклонения, факт,
// равный заявке, в событие не пишется).
type ProducedItem struct {
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	Reason      string  `json:"reason,omitempty"`
}

// ProductionRecorded — по заказу зафиксирована отработка (лист журнала).
// Повторное событие полностью заменяет предыдущий набор отклонений.
type ProductionRecorded struct {
	SheetID    int64          `json:"sheet_id"`
	ByUsername string         `json:"by_username"`
	Items      []ProducedItem `json:"items"`
}

// ProductionCleared — отработка снята с заказа.
type ProductionCleared struct {
	ByUsername string `json:"by_username"`
}
