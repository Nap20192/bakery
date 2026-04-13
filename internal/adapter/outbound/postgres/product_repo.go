package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	db "bakery/internal/adapter/outbound/db"
	"bakery/internal/domain/product"
)

type ProductRepository struct {
	q *db.Queries
}

func NewProductRepository(q *db.Queries) *ProductRepository {
	return &ProductRepository{q: q}
}

func (r *ProductRepository) Save(ctx context.Context, p *product.Product) error {
	if _, err := r.q.GetProductByID(ctx, p.ID()); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("product repo: probe: %w", err)
		}
		if _, err := r.q.CreateProduct(ctx, db.CreateProductParams{
			ID:     p.ID(),
			IikoID: p.IikoID(),
			Name:   p.Name(),
			Unit:   string(p.Unit()),
		}); err != nil {
			return fmt.Errorf("product repo: create: %w", err)
		}
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id product.ID) (*product.Product, error) {
	row, err := r.q.GetProductByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, product.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("product repo: get by id: %w", err)
	}
	return toProduct(row), nil
}

func (r *ProductRepository) GetByIikoID(ctx context.Context, iikoID string) (*product.Product, error) {
	id := iikoID
	row, err := r.q.GetProductByIikoID(ctx, &id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, product.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("product repo: get by iiko id: %w", err)
	}
	return toProduct(row), nil
}

func (r *ProductRepository) List(ctx context.Context) ([]*product.Product, error) {
	rows, err := r.q.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("product repo: list: %w", err)
	}
	out := make([]*product.Product, 0, len(rows))
	for _, row := range rows {
		out = append(out, toProduct(row))
	}
	return out, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id product.ID) error {
	if err := r.q.DeleteProduct(ctx, id); err != nil {
		return fmt.Errorf("product repo: delete: %w", err)
	}
	return nil
}

func toProduct(row db.Product) *product.Product {
	return product.Restore(row.ID, row.IikoID, row.Name, product.Unit(row.Unit), row.CreatedAt, row.UpdatedAt)
}
