package eventsourced

import (
	"fmt"
	"math"
	"strings"
	"time"

	"bakery/internal/pkg/apperr"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
)

// Ошибки инвариантов агрегата. Тексты совпадают со старым usecase-слоем,
// чтобы поведение API при переводе не изменилось.
var (
	ErrCancelled           = apperr.Conflict("order.cancelled", "Отменённый заказ нельзя изменить. Сначала восстановите его.")
	ErrNoItems             = apperr.Invalid("order.no_items", "Заказ должен содержать хотя бы одну позицию.")
	ErrDuplicateItem       = apperr.Invalid("order.duplicate_item", "Позиция добавлена повторно.")
	ErrBadQuantity         = apperr.Invalid("order.bad_quantity", "Количество должно быть положительным числом.")
	ErrProductionNoChanges = apperr.Invalid("order.production_no_changes", "Отработка без изменений не сохраняется: все значения совпадают с заявкой.")
	ErrProductionUnknown   = apperr.Invalid("order.production_item_unknown", "Позиции нет в заказе.")
)

// Order — event-sourced агрегат заказа: состояние строится только из потока
// событий (Transition), команды проверяют инварианты и добавляют события
// через aggregate.TrackChange. ID агрегата = номер заказа.
type Order struct {
	aggregate.Root

	Number            string
	FromDepartmentID  *int64
	ToDepartmentID    *int64
	CategoryID        *int64
	CreatedByUsername string
	CreatedAt         time.Time // из timestamp события Created
	FulfillmentDate   time.Time
	Items             []Item
	Comments          Comments
	Cancelled         bool

	// Отработка: только отклонения, ключ — имя позиции в нижнем регистре.
	ProductionSheetID *int64
	Produced          map[string]ProducedItem
}

// Register объявляет события, из которых собирается агрегат.
func (o *Order) Register(r aggregate.RegisterFunc) {
	r(&Created{}, &ItemsUpdated{}, &Cancelled{}, &Restored{}, &ProductionRecorded{}, &ProductionCleared{})
}

// Transition — единственное место, где меняется состояние. Никаких проверок:
// история уже случилась, её нужно только применить.
func (o *Order) Transition(event eventsourcing.Event) {
	switch data := event.Data().(type) {
	case *Created:
		o.Number = data.Number
		o.FromDepartmentID = data.FromDepartmentID
		o.ToDepartmentID = data.ToDepartmentID
		o.CategoryID = data.CategoryID
		o.CreatedByUsername = data.CreatedByUsername
		o.CreatedAt = event.Timestamp()
		o.FulfillmentDate = data.FulfillmentDate
		o.Items = data.Items
		o.Comments = data.Comments
	case *ItemsUpdated:
		o.Items = data.Items
		o.FulfillmentDate = data.FulfillmentDate
		o.Comments = data.Comments
	case *Cancelled:
		o.Cancelled = true
	case *Restored:
		o.Cancelled = false
	case *ProductionRecorded:
		o.ProductionSheetID = &data.SheetID
		o.Produced = make(map[string]ProducedItem, len(data.Items))
		for _, item := range data.Items {
			o.Produced[itemKey(item.ProductName)] = item
		}
	case *ProductionCleared:
		o.ProductionSheetID = nil
		o.Produced = nil
	}
}

// NewOrder — команда создания. Номер и нормализованные даты приходят от
// командного сервиса (генерация номера — счётчик вне агрегата).
func NewOrder(data Created) (*Order, error) {
	if strings.TrimSpace(data.Number) == "" {
		return nil, fmt.Errorf("order number is required")
	}
	if err := validateItems(data.Items); err != nil {
		return nil, err
	}
	order := &Order{}
	if err := order.SetID(data.Number); err != nil {
		return nil, fmt.Errorf("set aggregate id: %w", err)
	}
	aggregate.TrackChange(order, &data)
	return order, nil
}

// UpdateItems — команда изменения состава/даты/комментариев.
func (o *Order) UpdateItems(data ItemsUpdated) error {
	if o.Cancelled {
		return ErrCancelled
	}
	if err := validateItems(data.Items); err != nil {
		return err
	}
	aggregate.TrackChange(o, &data)
	return nil
}

// Cancel — мягкая отмена; повторная отмена — no-op без события.
func (o *Order) Cancel(byUsername string) {
	if o.Cancelled {
		return
	}
	aggregate.TrackChange(o, &Cancelled{ByUsername: strings.TrimSpace(byUsername)})
}

// Restore снимает отмену; активный заказ — no-op без события.
func (o *Order) Restore(byUsername string) {
	if !o.Cancelled {
		return
	}
	aggregate.TrackChange(o, &Restored{ByUsername: strings.TrimSpace(byUsername)})
}

// RecordProduction фиксирует отработку: только отклонения от заявки, позиции
// должны существовать в заказе. Полный набор отклонений заменяет предыдущий.
func (o *Order) RecordProduction(sheetID int64, byUsername string, items []ProducedItem) error {
	if o.Cancelled {
		return apperr.Conflict("order.production_cancelled", fmt.Sprintf("Заказ %s отменён — отработка невозможна.", o.Number))
	}
	byName := make(map[string]Item, len(o.Items))
	for _, item := range o.Items {
		byName[itemKey(item.ProductName)] = item
	}

	deviations := make([]ProducedItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, produced := range items {
		key := itemKey(produced.ProductName)
		if _, ok := seen[key]; ok {
			return ErrDuplicateItem
		}
		seen[key] = struct{}{}
		orderItem, ok := byName[key]
		if !ok {
			return ErrProductionUnknown
		}
		if produced.Quantity < 0 || math.IsNaN(produced.Quantity) || math.IsInf(produced.Quantity, 0) {
			return ErrBadQuantity
		}
		// Факт, равный заявке, отклонением не является.
		if produced.Quantity == orderItem.Quantity+orderItem.ReservedQuantity {
			continue
		}
		deviations = append(deviations, ProducedItem{
			ProductName: orderItem.ProductName,
			Quantity:    produced.Quantity,
			Reason:      strings.TrimSpace(produced.Reason),
		})
	}
	if len(deviations) == 0 {
		return ErrProductionNoChanges
	}
	aggregate.TrackChange(o, &ProductionRecorded{SheetID: sheetID, ByUsername: strings.TrimSpace(byUsername), Items: deviations})
	return nil
}

// ClearProduction снимает отработку; если её нет — no-op без события.
func (o *Order) ClearProduction(byUsername string) {
	if len(o.Produced) == 0 {
		return
	}
	aggregate.TrackChange(o, &ProductionCleared{ByUsername: strings.TrimSpace(byUsername)})
}

func validateItems(items []Item) error {
	if len(items) == 0 {
		return ErrNoItems
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := itemKey(item.ProductName)
		if strings.TrimSpace(item.ProductName) == "" {
			return ErrNoItems
		}
		if _, ok := seen[key]; ok {
			return ErrDuplicateItem
		}
		seen[key] = struct{}{}
		total := item.Quantity + item.ReservedQuantity
		if total <= 0 || math.IsNaN(total) || math.IsInf(total, 0) || item.Quantity < 0 || item.ReservedQuantity < 0 {
			return ErrBadQuantity
		}
	}
	return nil
}

func itemKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
