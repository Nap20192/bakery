package techcard

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain/ingredient"
	"bakery/internal/domain/product"
)

type ID = uuid.UUID
type ItemID = uuid.UUID

var (
	ErrInvalidYield   = errors.New("techcard: yield_qty must be > 0")
	ErrInvalidGross   = errors.New("techcard: gross_qty must be > 0")
	ErrInvalidNet     = errors.New("techcard: net_qty must be > 0")
	ErrInvalidSource  = errors.New("techcard: item must have exactly one source")
	ErrItemNotFound   = errors.New("techcard: item not found")
	ErrNotFound       = errors.New("techcard: not found")
)

// ItemSource — либо ингредиент, либо полуфабрикат (суб-продукт). XOR.
type ItemSource struct {
	IngredientID *ingredient.ID
	SubProductID *product.ID
}

func (s ItemSource) Valid() bool {
	return (s.IngredientID != nil) != (s.SubProductID != nil)
}

type Item struct {
	id         ItemID
	techCardID ID
	source     ItemSource
	grossQty   float64
	netQty     float64
	unit       product.Unit
	createdAt  time.Time
	updatedAt  time.Time
}

func newItem(techCardID ID, source ItemSource, grossQty, netQty float64, unit product.Unit) (*Item, error) {
	if !source.Valid() {
		return nil, ErrInvalidSource
	}
	if grossQty <= 0 {
		return nil, ErrInvalidGross
	}
	if netQty <= 0 {
		return nil, ErrInvalidNet
	}
	now := time.Now()
	return &Item{
		id:         uuid.New(),
		techCardID: techCardID,
		source:     source,
		grossQty:   grossQty,
		netQty:     netQty,
		unit:       unit,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func RestoreItem(id ItemID, techCardID ID, source ItemSource, grossQty, netQty float64, unit product.Unit, createdAt, updatedAt time.Time) *Item {
	return &Item{id: id, techCardID: techCardID, source: source, grossQty: grossQty, netQty: netQty, unit: unit, createdAt: createdAt, updatedAt: updatedAt}
}

func (i *Item) ID() ItemID         { return i.id }
func (i *Item) TechCardID() ID     { return i.techCardID }
func (i *Item) Source() ItemSource { return i.source }
func (i *Item) GrossQty() float64  { return i.grossQty }
func (i *Item) NetQty() float64    { return i.netQty }
func (i *Item) Unit() product.Unit { return i.unit }

type TechCard struct {
	id        ID
	productID product.ID
	yieldQty  float64
	yieldUnit product.Unit
	items     []*Item
	createdAt time.Time
	updatedAt time.Time
}

func New(productID product.ID, yieldQty float64, yieldUnit product.Unit) (*TechCard, error) {
	if yieldQty <= 0 {
		return nil, ErrInvalidYield
	}
	if !yieldUnit.Valid() {
		return nil, product.ErrInvalidUnit
	}
	now := time.Now()
	return &TechCard{
		id:        uuid.New(),
		productID: productID,
		yieldQty:  yieldQty,
		yieldUnit: yieldUnit,
		items:     make([]*Item, 0),
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Restore(id ID, productID product.ID, yieldQty float64, yieldUnit product.Unit, items []*Item, createdAt, updatedAt time.Time) *TechCard {
	return &TechCard{id: id, productID: productID, yieldQty: yieldQty, yieldUnit: yieldUnit, items: items, createdAt: createdAt, updatedAt: updatedAt}
}

func (tc *TechCard) ID() ID                 { return tc.id }
func (tc *TechCard) ProductID() product.ID  { return tc.productID }
func (tc *TechCard) YieldQty() float64      { return tc.yieldQty }
func (tc *TechCard) YieldUnit() product.Unit { return tc.yieldUnit }
func (tc *TechCard) Items() []*Item         { return tc.items }
func (tc *TechCard) CreatedAt() time.Time   { return tc.createdAt }
func (tc *TechCard) UpdatedAt() time.Time   { return tc.updatedAt }

func (tc *TechCard) AddItem(source ItemSource, grossQty, netQty float64, unit product.Unit) (*Item, error) {
	item, err := newItem(tc.id, source, grossQty, netQty, unit)
	if err != nil {
		return nil, err
	}
	tc.items = append(tc.items, item)
	tc.updatedAt = time.Now()
	return item, nil
}

func (tc *TechCard) RemoveItem(itemID ItemID) error {
	for i, item := range tc.items {
		if item.id == itemID {
			tc.items = append(tc.items[:i], tc.items[i+1:]...)
			tc.updatedAt = time.Now()
			return nil
		}
	}
	return ErrItemNotFound
}

type Repository interface {
	Save(ctx context.Context, tc *TechCard) error
	GetByID(ctx context.Context, id ID) (*TechCard, error)
	GetByProductID(ctx context.Context, productID product.ID) (*TechCard, error)
	List(ctx context.Context) ([]*TechCard, error)
	Delete(ctx context.Context, id ID) error
}
