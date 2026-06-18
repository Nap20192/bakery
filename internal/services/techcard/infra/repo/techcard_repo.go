// Package techcardrepo is the persistence adapter of the techcard service. It
// assembles a tech card from the iiko snapshot (products + assembly/prepared
// charts) stored via sqlc.
package techcardrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"
	techcarddomain "bakery/internal/services/techcard/domain"
	techcarduc "bakery/internal/services/techcard/usecase/techcard"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TechCardRepository struct {
	queries *sqlc.Queries
}

var _ techcarduc.Repository = (*TechCardRepository)(nil)

func New(queries *sqlc.Queries) *TechCardRepository {
	return &TechCardRepository{queries: queries}
}

func (r *TechCardRepository) GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error) {
	product, err := r.queries.GetIikoProductByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return techcarddomain.TechCard{}, fmt.Errorf("product with code %s not found", code)
		}
		return techcarddomain.TechCard{}, fmt.Errorf("get product by code: %w", err)
	}

	card := techcarddomain.TechCard{
		Code:     product.Code,
		Name:     product.Name,
		Unit:     product.MeasureUnit,
		Products: make(map[string]techcarddomain.TechCardProduct),
	}
	if product.Type != nil {
		card.Type = *product.Type
	}

	dateParam := pgtype.Date{Time: date, Valid: true}
	if err := r.attachAssembly(ctx, &card, product.ID, dateParam); err != nil {
		return techcarddomain.TechCard{}, err
	}
	if card.Assembly == nil {
		if err := r.attachPrepared(ctx, &card, product.ID, dateParam); err != nil {
			return techcarddomain.TechCard{}, err
		}
	}
	return card, nil
}

func (r *TechCardRepository) attachAssembly(ctx context.Context, card *techcarddomain.TechCard, productID string, date pgtype.Date) error {
	row, err := r.queries.GetActiveAssemblyChartFullByProductID(ctx, sqlc.GetActiveAssemblyChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          date,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get active assembly chart: %w", err)
	}

	var dto iiko.AssemblyChartDto
	if err := json.Unmarshal([]byte(row.RawJson), &dto); err != nil {
		return fmt.Errorf("decode assembly chart raw json: %w", err)
	}
	card.Assembly = &dto
	for _, item := range dto.Items {
		r.addProduct(ctx, card, item.ProductID)
	}
	return nil
}

func (r *TechCardRepository) attachPrepared(ctx context.Context, card *techcarddomain.TechCard, productID string, date pgtype.Date) error {
	row, err := r.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          date,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get active prepared chart: %w", err)
	}

	var dto iiko.PreparedChartDto
	if err := json.Unmarshal([]byte(row.RawJson), &dto); err != nil {
		return fmt.Errorf("decode prepared chart raw json: %w", err)
	}
	card.Prepared = &dto
	for _, item := range dto.Items {
		r.addProduct(ctx, card, item.ProductID)
	}
	return nil
}

func (r *TechCardRepository) addProduct(ctx context.Context, card *techcarddomain.TechCard, productID string) {
	if productID == "" {
		return
	}
	if _, ok := card.Products[productID]; ok {
		return
	}
	product, err := r.queries.GetIikoProductByID(ctx, productID)
	if err != nil {
		card.Products[productID] = techcarddomain.TechCardProduct{
			ID:   productID,
			Name: "product not found in iiko_products",
		}
		return
	}
	item := techcarddomain.TechCardProduct{
		ID:   product.ID,
		Code: product.Code,
		Name: product.Name,
		Unit: product.MeasureUnit,
	}
	if product.Type != nil {
		item.Type = *product.Type
	}
	card.Products[productID] = item
}
