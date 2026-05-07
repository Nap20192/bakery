package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	techcarddomain "bakery/internal/domain/techcard"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TechCardService struct {
	queries *sqlc.Queries
}

func NewTechCardService(queries *sqlc.Queries) *TechCardService {
	return &TechCardService{queries: queries}
}

func (s *TechCardService) GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return techcarddomain.TechCard{}, fmt.Errorf("code is required")
	}
	if date.IsZero() {
		date = time.Now().UTC()
	}
	product, err := s.queries.GetIikoProductByCode(ctx, code)
	if err != nil {
		if err == pgx.ErrNoRows {
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
	if err := s.attachAssembly(ctx, &card, product.ID, dateParam); err != nil {
		return techcarddomain.TechCard{}, err
	}
	if card.Assembly == nil {
		if err := s.attachPrepared(ctx, &card, product.ID, dateParam); err != nil {
			return techcarddomain.TechCard{}, err
		}
	}

	return card, nil
}

func (s *TechCardService) attachAssembly(ctx context.Context, card *techcarddomain.TechCard, productID string, date pgtype.Date) error {
	row, err := s.queries.GetActiveAssemblyChartFullByProductID(ctx, sqlc.GetActiveAssemblyChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          date,
	})
	if err == pgx.ErrNoRows {
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
		s.addProduct(ctx, card, item.ProductID)
	}
	return nil
}

func (s *TechCardService) attachPrepared(ctx context.Context, card *techcarddomain.TechCard, productID string, date pgtype.Date) error {
	row, err := s.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          date,
	})
	if err == pgx.ErrNoRows {
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
		s.addProduct(ctx, card, item.ProductID)
	}
	return nil
}

func (s *TechCardService) addProduct(ctx context.Context, card *techcarddomain.TechCard, productID string) {
	if productID == "" {
		return
	}
	if _, ok := card.Products[productID]; ok {
		return
	}
	product, err := s.queries.GetIikoProductByID(ctx, productID)
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
