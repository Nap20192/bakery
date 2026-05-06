package app

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
)

type MonitorService struct {
	queries *sqlc.Queries
}

func NewMonitorService(queries *sqlc.Queries) *MonitorService {
	return &MonitorService{queries: queries}
}

func (s *MonitorService) GetIngredientsByGroup(ctx context.Context, group domain.Group, order domain.Order) (domain.GroupIngredientsReport, error) {
	report := domain.GroupIngredientsReport{Group: group}
	if s == nil || s.queries == nil {
		return report, fmt.Errorf("monitor service is not initialized")
	}
	ingredient, err := s.resolveIngredient(ctx, group)
	if err != nil {
		return report, err
	}
	report.Ingredient = domain.GroupIngredientUsage{
		ProductID:   ingredient.ID,
		ProductCode: ingredient.Code,
		ProductName: ingredient.Name,
		Unit:        ingredient.MeasureUnit,
	}

	orderDate := order.CreatedAt
	if orderDate.IsZero() {
		orderDate = time.Now().UTC()
	}
	orderDateText := orderDate.Format("2006-01-02")

	for _, orderItem := range order.Items {
		product, err := s.queries.GetIikoProductByCode(ctx, orderItem.Code)
		if err != nil {
			if err == sql.ErrNoRows {
				report.Warnings = append(report.Warnings, fmt.Sprintf("product with code %s not found", orderItem.Code))
				continue
			}
			return report, fmt.Errorf("get product by code %s: %w", orderItem.Code, err)
		}

		chart, err := s.queries.GetActiveAssemblyChartByProductID(ctx, sqlc.GetActiveAssemblyChartByProductIDParams{
			AssembledProductID: product.ID,
			OrderDate:          orderDateText,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				report.Warnings = append(report.Warnings, fmt.Sprintf("assembly chart for product %s (%s) not found", orderItem.ProductName, orderItem.Code))
				continue
			}
			return report, fmt.Errorf("get assembly chart for product %s: %w", orderItem.Code, err)
		}

		chartItems, err := s.queries.ListAssemblyChartItemsByChartID(ctx, chart.ID)
		if err != nil {
			return report, fmt.Errorf("list chart items for chart %s: %w", chart.ID, err)
		}

		scale := orderItem.Quantity
		if chart.AssembledAmount > 0 {
			scale = orderItem.Quantity / chart.AssembledAmount
		}

		breakdown := domain.GroupIngredientDishBreakdown{
			OrderItemCode:     orderItem.Code,
			OrderItemName:     orderItem.ProductName,
			OrderItemQuantity: orderItem.Quantity,
		}

		for _, chartItem := range chartItems {
			if chartItem.ProductID != ingredient.ID {
				continue
			}

			used := chartItem.AmountOut * scale
			breakdown.IngredientQuantity += used
			report.Ingredient.Quantity += used
		}

		if breakdown.IngredientQuantity > 0 {
			report.Breakdown = append(report.Breakdown, breakdown)
		}
	}

	if report.Ingredient.Quantity > 0 {
		for i := range report.Breakdown {
			report.Breakdown[i].ProportionOfTotal = report.Breakdown[i].IngredientQuantity / report.Ingredient.Quantity
		}
	}
	sort.Slice(report.Breakdown, func(i, j int) bool {
		return report.Breakdown[i].IngredientQuantity > report.Breakdown[j].IngredientQuantity
	})

	return report, nil
}

func (s *MonitorService) resolveIngredient(ctx context.Context, group domain.Group) (sqlc.GetIikoProductByIDRow, error) {
	if group.ID != "" {
		product, err := s.queries.GetIikoProductByID(ctx, group.ID)
		if err == nil {
			return product, nil
		}
		if err != sql.ErrNoRows {
			return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("get ingredient by id %s: %w", group.ID, err)
		}
	}
	if group.Code != "" {
		product, err := s.queries.GetIikoProductByCode(ctx, group.Code)
		if err == nil {
			return sqlc.GetIikoProductByIDRow(product), nil
		}
		if err != sql.ErrNoRows {
			return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("get ingredient by code %s: %w", group.Code, err)
		}
	}
	if group.Name != "" {
		products, err := s.queries.GetIikoProductsByName(ctx, group.Name)
		if err != nil {
			return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("get ingredient by name %s: %w", group.Name, err)
		}
		if len(products) == 1 {
			return sqlc.GetIikoProductByIDRow(products[0]), nil
		}
		if len(products) > 1 {
			return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("ingredient name %q is ambiguous: %d products matched", group.Name, len(products))
		}
	}
	return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("ingredient not found by group.id/group.code/group.name")
}
