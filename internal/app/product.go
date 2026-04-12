package app

import (
	"context"
	"fmt"

	"bakery/internal/domain/product"
)

type ProductService struct {
	products product.Repository
}

func NewProductService(products product.Repository) *ProductService {
	return &ProductService{products: products}
}

type CreateProductInput struct {
	Name   string
	Unit   string
	IikoID string
}

func (s *ProductService) Create(ctx context.Context, in CreateProductInput) (*product.Product, error) {
	var iikoID *string
	if in.IikoID != "" {
		v := in.IikoID
		iikoID = &v
	}
	p, err := product.New(in.Name, product.Unit(in.Unit), iikoID)
	if err != nil {
		return nil, err
	}
	if err := s.products.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("product: save: %w", err)
	}
	return p, nil
}

func (s *ProductService) GetByID(ctx context.Context, id product.ID) (*product.Product, error) {
	return s.products.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context) ([]*product.Product, error) {
	return s.products.List(ctx)
}

func (s *ProductService) Delete(ctx context.Context, id product.ID) error {
	return s.products.Delete(ctx, id)
}
