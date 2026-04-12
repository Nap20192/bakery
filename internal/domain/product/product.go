package product

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ID = uuid.UUID

type Unit string

const (
	UnitPiece Unit = "шт"
	UnitKg    Unit = "кг"
	UnitL     Unit = "л"
	UnitG     Unit = "г"
	UnitMl    Unit = "мл"
)

func (u Unit) Valid() bool {
	switch u {
	case UnitPiece, UnitKg, UnitL, UnitG, UnitMl:
		return true
	}
	return false
}

var (
	ErrNameRequired = errors.New("product: name is required")
	ErrInvalidUnit  = errors.New("product: invalid unit")
	ErrNotFound     = errors.New("product: not found")
)

type Product struct {
	id        ID
	iikoID    *string
	name      string
	unit      Unit
	createdAt time.Time
	updatedAt time.Time
}

func New(name string, unit Unit, iikoID *string) (*Product, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	if !unit.Valid() {
		return nil, ErrInvalidUnit
	}
	now := time.Now()
	return &Product{
		id:        uuid.New(),
		iikoID:    iikoID,
		name:      name,
		unit:      unit,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Restore(id ID, iikoID *string, name string, unit Unit, createdAt, updatedAt time.Time) *Product {
	return &Product{id: id, iikoID: iikoID, name: name, unit: unit, createdAt: createdAt, updatedAt: updatedAt}
}

func (p *Product) ID() ID               { return p.id }
func (p *Product) IikoID() *string      { return p.iikoID }
func (p *Product) Name() string         { return p.name }
func (p *Product) Unit() Unit           { return p.unit }
func (p *Product) CreatedAt() time.Time { return p.createdAt }
func (p *Product) UpdatedAt() time.Time { return p.updatedAt }

type Repository interface {
	Save(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id ID) (*Product, error)
	GetByIikoID(ctx context.Context, iikoID string) (*Product, error)
	List(ctx context.Context) ([]*Product, error)
	Delete(ctx context.Context, id ID) error
}
