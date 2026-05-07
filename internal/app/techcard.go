package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bakery/internal/domain"
	"bakery/internal/iiko"
	"bakery/internal/repo/sqlc"
)

type TechCardService struct {
	queries *sqlc.Queries
}

func NewTechCardService(queries *sqlc.Queries) *TechCardService {
	return &TechCardService{queries: queries}
}

func (s *TechCardService) GetByCode(ctx context.Context, code string, date time.Time) (domain.TechCard, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return domain.TechCard{}, fmt.Errorf("code is required")
	}
	if date.IsZero() {
		date = time.Now().UTC()
	}
	product, err := s.queries.GetIikoProductByCode(ctx, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.TechCard{}, fmt.Errorf("product with code %s not found", code)
		}
		return domain.TechCard{}, fmt.Errorf("get product by code: %w", err)
	}

	card := domain.TechCard{
		ProductID: product.ID,
		Code:      product.Code,
		Name:      product.Name,
		Unit:      product.MeasureUnit,
		Products:  make(map[string]domain.TechCardProduct),
	}
	if product.Type != nil {
		card.Type = *product.Type
	}

	dateText := date.Format("2006-01-02")
	if err := s.attachAssembly(ctx, &card, product.ID, dateText); err != nil {
		return domain.TechCard{}, err
	}
	if card.Assembly == nil {
		if err := s.attachPrepared(ctx, &card, product.ID, dateText); err != nil {
			return domain.TechCard{}, err
		}
	}

	return card, nil
}

func (s *TechCardService) attachAssembly(ctx context.Context, card *domain.TechCard, productID string, date string) error {
	row, err := s.queries.GetActiveAssemblyChartFullByProductID(ctx, sqlc.GetActiveAssemblyChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          date,
	})
	if err == sql.ErrNoRows {
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

func (s *TechCardService) attachPrepared(ctx context.Context, card *domain.TechCard, productID string, date string) error {
	row, err := s.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          date,
	})
	if err == sql.ErrNoRows {
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

func (s *TechCardService) addProduct(ctx context.Context, card *domain.TechCard, productID string) {
	if productID == "" {
		return
	}
	if _, ok := card.Products[productID]; ok {
		return
	}
	product, err := s.queries.GetIikoProductByID(ctx, productID)
	if err != nil {
		card.Products[productID] = domain.TechCardProduct{
			ID:   productID,
			Name: "product not found in iiko_products",
		}
		return
	}
	item := domain.TechCardProduct{
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
