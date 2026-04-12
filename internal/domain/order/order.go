package order

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain/client"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/product"
)

type ID = uuid.UUID
type ItemID = uuid.UUID

type Status string

const (
	StatusNew       Status = "new"
	StatusParsed    Status = "parsed"
	StatusConfirmed Status = "confirmed"
	StatusProcessed Status = "processed"
	StatusCancelled Status = "cancelled"
)

var allowedTransitions = map[Status][]Status{
	StatusNew:       {StatusParsed, StatusCancelled},
	StatusParsed:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed: {StatusProcessed, StatusCancelled},
	StatusProcessed: {},
	StatusCancelled: {},
}

func (s Status) CanTransitionTo(next Status) bool {
	for _, a := range allowedTransitions[s] {
		if a == next {
			return true
		}
	}
	return false
}

var (
	ErrOrderDateRequired = errors.New("order: order_date is required")
	ErrInvalidQty        = errors.New("order: quantity must be > 0")
	ErrInvalidStatus     = errors.New("order: invalid status transition")
	ErrLocked            = errors.New("order: cannot modify confirmed or processed order")
	ErrItemNotFound      = errors.New("order: item not found")
	ErrNotFound          = errors.New("order: not found")
)

// TelegramSource — откуда пришёл заказ в Telegram.
type TelegramSource struct {
	ChatID    int64
	MessageID int64
}

type Item struct {
	id        ItemID
	orderID   ID
	productID product.ID
	quantity  float64
	unit      product.Unit
	createdAt time.Time
	updatedAt time.Time
}

func newItem(orderID ID, productID product.ID, quantity float64, unit product.Unit) (*Item, error) {
	if quantity <= 0 {
		return nil, ErrInvalidQty
	}
	now := time.Now()
	return &Item{
		id:        uuid.New(),
		orderID:   orderID,
		productID: productID,
		quantity:  quantity,
		unit:      unit,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RestoreItem(id ItemID, orderID ID, productID product.ID, quantity float64, unit product.Unit, createdAt, updatedAt time.Time) *Item {
	return &Item{id: id, orderID: orderID, productID: productID, quantity: quantity, unit: unit, createdAt: createdAt, updatedAt: updatedAt}
}

func (i *Item) ID() ItemID           { return i.id }
func (i *Item) OrderID() ID          { return i.orderID }
func (i *Item) ProductID() product.ID { return i.productID }
func (i *Item) Quantity() float64    { return i.quantity }
func (i *Item) Unit() product.Unit   { return i.unit }

type Order struct {
	id        ID
	clientID  client.ID
	kitchenID *kitchen.ID
	telegram  *TelegramSource
	orderDate time.Time
	status    Status
	rawText   string
	note      string
	items     []*Item
	createdAt time.Time
	updatedAt time.Time
}

func New(clientID client.ID, kitchenID *kitchen.ID, orderDate time.Time, telegram *TelegramSource, rawText, note string) (*Order, error) {
	if orderDate.IsZero() {
		return nil, ErrOrderDateRequired
	}
	now := time.Now()
	return &Order{
		id:        uuid.New(),
		clientID:  clientID,
		kitchenID: kitchenID,
		telegram:  telegram,
		orderDate: orderDate,
		status:    StatusNew,
		rawText:   rawText,
		note:      note,
		items:     make([]*Item, 0),
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Restore(id ID, clientID client.ID, kitchenID *kitchen.ID, telegram *TelegramSource, orderDate time.Time, status Status, rawText, note string, items []*Item, createdAt, updatedAt time.Time) *Order {
	return &Order{id: id, clientID: clientID, kitchenID: kitchenID, telegram: telegram, orderDate: orderDate, status: status, rawText: rawText, note: note, items: items, createdAt: createdAt, updatedAt: updatedAt}
}

func (o *Order) ID() ID                  { return o.id }
func (o *Order) ClientID() client.ID     { return o.clientID }
func (o *Order) KitchenID() *kitchen.ID  { return o.kitchenID }
func (o *Order) Telegram() *TelegramSource { return o.telegram }
func (o *Order) OrderDate() time.Time    { return o.orderDate }
func (o *Order) Status() Status          { return o.status }
func (o *Order) RawText() string         { return o.rawText }
func (o *Order) Note() string            { return o.note }
func (o *Order) Items() []*Item          { return o.items }
func (o *Order) CreatedAt() time.Time    { return o.createdAt }
func (o *Order) UpdatedAt() time.Time    { return o.updatedAt }

func (o *Order) Transition(next Status) error {
	if !o.status.CanTransitionTo(next) {
		return ErrInvalidStatus
	}
	o.status = next
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) AssignKitchen(kitchenID kitchen.ID) error {
	if o.status == StatusProcessed || o.status == StatusCancelled {
		return ErrLocked
	}
	o.kitchenID = &kitchenID
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) AddItem(productID product.ID, quantity float64, unit product.Unit) (*Item, error) {
	if o.status == StatusConfirmed || o.status == StatusProcessed {
		return nil, ErrLocked
	}
	item, err := newItem(o.id, productID, quantity, unit)
	if err != nil {
		return nil, err
	}
	o.items = append(o.items, item)
	o.updatedAt = time.Now()
	return item, nil
}

func (o *Order) RemoveItem(itemID ItemID) error {
	if o.status == StatusConfirmed || o.status == StatusProcessed {
		return ErrLocked
	}
	for i, item := range o.items {
		if item.id == itemID {
			o.items = append(o.items[:i], o.items[i+1:]...)
			o.updatedAt = time.Now()
			return nil
		}
	}
	return ErrItemNotFound
}

type Filter struct {
	ClientID  *client.ID
	KitchenID *kitchen.ID
	Status    *Status
	DateFrom  *time.Time
	DateTo    *time.Time
}

type Repository interface {
	Save(ctx context.Context, o *Order) error
	GetByID(ctx context.Context, id ID) (*Order, error)
	GetByTelegram(ctx context.Context, chatID, messageID int64) (*Order, error)
	List(ctx context.Context, f Filter) ([]*Order, error)
	Delete(ctx context.Context, id ID) error
}
