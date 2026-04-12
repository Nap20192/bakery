package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	db "bakery/internal/adapter/outbound/db"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/user"
)

type KitchenRepository struct {
	q *db.Queries
}

func NewKitchenRepository(q *db.Queries) *KitchenRepository {
	return &KitchenRepository{q: q}
}

func (r *KitchenRepository) Save(ctx context.Context, k *kitchen.Kitchen) error {
	_, err := r.q.GetKitchenByID(ctx, k.ID())
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := r.q.CreateKitchen(ctx, db.CreateKitchenParams{
			ID:     k.ID(),
			UserID: k.UserID(),
			Name:   k.Name(),
		}); err != nil {
			return fmt.Errorf("kitchen repo: create: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("kitchen repo: probe: %w", err)
	}
	if _, err := r.q.UpdateKitchen(ctx, db.UpdateKitchenParams{
		ID:   k.ID(),
		Name: k.Name(),
	}); err != nil {
		return fmt.Errorf("kitchen repo: update: %w", err)
	}
	return nil
}

func (r *KitchenRepository) GetByID(ctx context.Context, id kitchen.ID) (*kitchen.Kitchen, error) {
	row, err := r.q.GetKitchenByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kitchen.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("kitchen repo: get by id: %w", err)
	}
	return toKitchen(row), nil
}

func (r *KitchenRepository) GetByUserID(ctx context.Context, userID user.ID) (*kitchen.Kitchen, error) {
	uid := userID
	row, err := r.q.GetKitchenByUserID(ctx, &uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kitchen.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("kitchen repo: get by user id: %w", err)
	}
	return toKitchen(row), nil
}

func (r *KitchenRepository) List(ctx context.Context) ([]*kitchen.Kitchen, error) {
	rows, err := r.q.ListKitchens(ctx)
	if err != nil {
		return nil, fmt.Errorf("kitchen repo: list: %w", err)
	}
	out := make([]*kitchen.Kitchen, 0, len(rows))
	for _, row := range rows {
		out = append(out, toKitchen(row))
	}
	return out, nil
}

func (r *KitchenRepository) Delete(ctx context.Context, id kitchen.ID) error {
	if err := r.q.DeleteKitchen(ctx, id); err != nil {
		return fmt.Errorf("kitchen repo: delete: %w", err)
	}
	return nil
}

func toKitchen(row db.Kitchen) *kitchen.Kitchen {
	return kitchen.Restore(row.ID, row.UserID, row.Name, row.CreatedAt, row.UpdatedAt)
}
