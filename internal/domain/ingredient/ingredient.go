package ingredient

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain/product"
)

type ID = uuid.UUID

var (
	ErrNameRequired = errors.New("ingredient: name is required")
	ErrInvalidUnit  = errors.New("ingredient: invalid unit")
	ErrNotFound     = errors.New("ingredient: not found")
)

type Ingredient struct {
	id        ID
	iikoID    *string
	name      string
	unit      product.Unit
	createdAt time.Time
	updatedAt time.Time
}

func New(name string, unit product.Unit, iikoID *string) (*Ingredient, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	if !unit.Valid() {
		return nil, ErrInvalidUnit
	}
	now := time.Now()
	return &Ingredient{
		id:        uuid.New(),
		iikoID:    iikoID,
		name:      name,
		unit:      unit,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Restore(id ID, iikoID *string, name string, unit product.Unit, createdAt, updatedAt time.Time) *Ingredient {
	return &Ingredient{id: id, iikoID: iikoID, name: name, unit: unit, createdAt: createdAt, updatedAt: updatedAt}
}

func (i *Ingredient) ID() ID               { return i.id }
func (i *Ingredient) IikoID() *string      { return i.iikoID }
func (i *Ingredient) Name() string         { return i.name }
func (i *Ingredient) Unit() product.Unit   { return i.unit }
func (i *Ingredient) CreatedAt() time.Time { return i.createdAt }
func (i *Ingredient) UpdatedAt() time.Time { return i.updatedAt }

type Repository interface {
	Save(ctx context.Context, ing *Ingredient) error
	GetByID(ctx context.Context, id ID) (*Ingredient, error)
	GetByIikoID(ctx context.Context, iikoID string) (*Ingredient, error)
	List(ctx context.Context) ([]*Ingredient, error)
	Delete(ctx context.Context, id ID) error
}
