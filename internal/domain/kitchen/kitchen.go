package kitchen

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain/user"
)

type ID = uuid.UUID

var (
	ErrNameRequired = errors.New("kitchen: name is required")
	ErrNotFound     = errors.New("kitchen: not found")
)

type Kitchen struct {
	id        ID
	userID    *user.ID
	name      string
	createdAt time.Time
	updatedAt time.Time
}

func New(name string, userID *user.ID) (*Kitchen, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	now := time.Now()
	return &Kitchen{
		id:        uuid.New(),
		userID:    userID,
		name:      name,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Restore(id ID, userID *user.ID, name string, createdAt, updatedAt time.Time) *Kitchen {
	return &Kitchen{id: id, userID: userID, name: name, createdAt: createdAt, updatedAt: updatedAt}
}

func (k *Kitchen) ID() ID               { return k.id }
func (k *Kitchen) UserID() *user.ID     { return k.userID }
func (k *Kitchen) Name() string         { return k.name }
func (k *Kitchen) CreatedAt() time.Time { return k.createdAt }
func (k *Kitchen) UpdatedAt() time.Time { return k.updatedAt }

func (k *Kitchen) Rename(name string) error {
	if name == "" {
		return ErrNameRequired
	}
	k.name = name
	k.updatedAt = time.Now()
	return nil
}

type Repository interface {
	Save(ctx context.Context, k *Kitchen) error
	GetByID(ctx context.Context, id ID) (*Kitchen, error)
	GetByUserID(ctx context.Context, userID user.ID) (*Kitchen, error)
	List(ctx context.Context) ([]*Kitchen, error)
	Delete(ctx context.Context, id ID) error
}
