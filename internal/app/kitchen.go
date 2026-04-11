package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"bakery/internal/domain"
	"bakery/internal/adapter/outbound/postgres"
)

type KitchenService struct {
	q *postgres.Queries
}

func NewKitchenService(q *postgres.Queries) *KitchenService {
	return &KitchenService{q: q}
}

type CreateKitchenInput struct {
	Name    string
	Address string
	IikoID  string
}

func (s *KitchenService) Create(ctx context.Context, in CreateKitchenInput) (domain.Kitchen, error) {
	if in.Name == "" {
		return domain.Kitchen{}, fmt.Errorf("kitchen name is required")
	}

	row, err := s.q.KitchenInsert(ctx, postgres.KitchenInsertParams{
		Name:    in.Name,
		Address: strPtr(in.Address),
		IikoID:  strPtr(in.IikoID),
	})
	if err != nil {
		return domain.Kitchen{}, fmt.Errorf("create kitchen: %w", err)
	}
	return kitchenToDomain(row), nil
}

func (s *KitchenService) GetByID(ctx context.Context, id uuid.UUID) (domain.Kitchen, error) {
	row, err := s.q.KitchenGetByID(ctx, id)
	if err != nil {
		return domain.Kitchen{}, fmt.Errorf("get kitchen %s: %w", id, err)
	}
	return kitchenToDomain(row), nil
}

func (s *KitchenService) GetByIikoID(ctx context.Context, iikoID string) (domain.Kitchen, error) {
	row, err := s.q.KitchenGetByIikoID(ctx, &iikoID)
	if err != nil {
		return domain.Kitchen{}, fmt.Errorf("get kitchen by iiko_id %s: %w", iikoID, err)
	}
	return kitchenToDomain(row), nil
}

func (s *KitchenService) List(ctx context.Context) ([]domain.Kitchen, error) {
	rows, err := s.q.KitchenList(ctx)
	if err != nil {
		return nil, fmt.Errorf("list kitchens: %w", err)
	}
	out := make([]domain.Kitchen, len(rows))
	for i, r := range rows {
		out[i] = kitchenToDomain(r)
	}
	return out, nil
}

type UpdateKitchenInput struct {
	ID      uuid.UUID
	Name    string
	Address string
	IikoID  string
}

func (s *KitchenService) Update(ctx context.Context, in UpdateKitchenInput) (domain.Kitchen, error) {
	if in.Name == "" {
		return domain.Kitchen{}, fmt.Errorf("kitchen name is required")
	}

	row, err := s.q.KitchenUpdate(ctx, postgres.KitchenUpdateParams{
		ID:      in.ID,
		Name:    in.Name,
		Address: strPtr(in.Address),
		IikoID:  strPtr(in.IikoID),
	})
	if err != nil {
		return domain.Kitchen{}, fmt.Errorf("update kitchen %s: %w", in.ID, err)
	}
	return kitchenToDomain(row), nil
}

func (s *KitchenService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.KitchenSoftDelete(ctx, id); err != nil {
		return fmt.Errorf("delete kitchen %s: %w", id, err)
	}
	return nil
}

func kitchenToDomain(r postgres.Kitchen) domain.Kitchen {
	return domain.Kitchen{
		ID:        r.ID,
		Name:      r.Name,
		Address:   derefStr(r.Address),
		IikoID:    derefStr(r.IikoID),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		DeletedAt: r.DeletedAt,
	}
}

