package app

import (
	"context"
	"fmt"

	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/user"
)

type KitchenService struct {
	kitchens kitchen.Repository
}

func NewKitchenService(kitchens kitchen.Repository) *KitchenService {
	return &KitchenService{kitchens: kitchens}
}

type CreateKitchenInput struct {
	Name   string
	UserID *user.ID
}

type UpdateKitchenInput struct {
	Name string
}

func (s *KitchenService) Create(ctx context.Context, in CreateKitchenInput) (*kitchen.Kitchen, error) {
	k, err := kitchen.New(in.Name, in.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.kitchens.Save(ctx, k); err != nil {
		return nil, fmt.Errorf("kitchen: save: %w", err)
	}
	return k, nil
}

func (s *KitchenService) GetByID(ctx context.Context, id kitchen.ID) (*kitchen.Kitchen, error) {
	return s.kitchens.GetByID(ctx, id)
}

func (s *KitchenService) List(ctx context.Context) ([]*kitchen.Kitchen, error) {
	return s.kitchens.List(ctx)
}

func (s *KitchenService) Update(ctx context.Context, id kitchen.ID, in UpdateKitchenInput) (*kitchen.Kitchen, error) {
	k, err := s.kitchens.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := k.Rename(in.Name); err != nil {
		return nil, err
	}
	if err := s.kitchens.Save(ctx, k); err != nil {
		return nil, fmt.Errorf("kitchen: save: %w", err)
	}
	return k, nil
}

func (s *KitchenService) Delete(ctx context.Context, id kitchen.ID) error {
	return s.kitchens.Delete(ctx, id)
}
