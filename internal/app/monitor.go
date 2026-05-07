package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxMonitorDepth = 12

type MonitorService struct {
	queries *sqlc.Queries
}

func NewMonitorService(queries *sqlc.Queries) *MonitorService {
	return &MonitorService{queries: queries}
}

func (s *MonitorService) GetIngredientsByCode(ctx context.Context, code string, order orderdomain.Order) (monitoringdomain.IngredientReport, error) {
	report := monitoringdomain.IngredientReport{}
	if s == nil || s.queries == nil {
		return report, fmt.Errorf("monitor service is not initialized")
	}
	ingredient, err := s.resolveIngredient(ctx, code)
	if err != nil {
		return report, err
	}
	report.Ingredient = monitoringdomain.IngredientUsage{
		ProductCode: ingredient.Code,
		ProductName: ingredient.Name,
		Unit:        monitorUnit(ingredient),
	}

	orderDate := order.CreatedAt
	if orderDate.IsZero() {
		orderDate = time.Now().UTC()
	}
	orderDateParam := pgDate(orderDate)

	for _, orderItem := range order.Items {
		breakdown := monitoringdomain.IngredientDishBreakdown{
			OrderItemCode:     orderItem.Code,
			OrderItemName:     orderItem.ProductName,
			OrderItemQuantity: orderItem.Quantity,
		}

		product, err := s.queries.GetIikoProductByCode(ctx, orderItem.Code)
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return report, fmt.Errorf("get product by code %s: %w", orderItem.Code, err)
		}

		used, err := s.ingredientUsageForProduct(ctx, product.ID, ingredient.ID, orderItem.Quantity, orderDateParam, map[string]bool{})
		if err != nil {
			return report, fmt.Errorf("calculate ingredient usage for product %s: %w", orderItem.Code, err)
		}
		breakdown.IngredientQuantity = used
		report.Ingredient.Quantity += used

		if breakdown.IngredientQuantity > 0 {
			report.Breakdown = append(report.Breakdown, breakdown)
		}
	}

	sort.Slice(report.Breakdown, func(i, j int) bool {
		return report.Breakdown[i].IngredientQuantity > report.Breakdown[j].IngredientQuantity
	})

	return report, nil
}

func monitorUnit(product sqlc.GetIikoProductByIDRow) string {
	unit := strings.TrimSpace(product.MeasureUnit)
	if unit != "" {
		return unit
	}
	if product.Type != nil && strings.EqualFold(*product.Type, "PREPARED") {
		return "кг"
	}
	return ""
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func (s *MonitorService) ingredientUsageForProduct(
	ctx context.Context,
	productID string,
	ingredientID string,
	amount float64,
	orderDate pgtype.Date,
	path map[string]bool,
) (float64, error) {
	if productID == "" || amount == 0 {
		return 0, nil
	}
	if productID == ingredientID {
		return amount, nil
	}
	if path[productID] || len(path) >= maxMonitorDepth {
		return 0, nil
	}

	nextPath := make(map[string]bool, len(path)+1)
	for key, value := range path {
		nextPath[key] = value
	}
	nextPath[productID] = true

	assembly, err := s.queries.GetActiveAssemblyChartByProductID(ctx, sqlc.GetActiveAssemblyChartByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          orderDate,
	})
	if err == nil {
		items, err := s.queries.ListAssemblyChartItemsByChartID(ctx, assembly.ID)
		if err != nil {
			return 0, fmt.Errorf("list assembly chart items %s: %w", assembly.ID, err)
		}
		scale := amount
		if assembly.AssembledAmount > 0 {
			scale = amount / assembly.AssembledAmount
		}

		var total float64
		for _, item := range items {
			childAmount := item.AmountIn * scale
			used, err := s.ingredientUsageForProduct(ctx, item.ProductID, ingredientID, childAmount, orderDate, nextPath)
			if err != nil {
				return 0, err
			}
			total += used
		}
		return total, nil
	}
	if err != pgx.ErrNoRows {
		return 0, fmt.Errorf("get active assembly chart: %w", err)
	}

	prepared, err := s.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          orderDate,
	})
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get active prepared chart: %w", err)
	}

	var dto iiko.PreparedChartDto
	if err := json.Unmarshal([]byte(prepared.RawJson), &dto); err != nil {
		return 0, fmt.Errorf("decode prepared chart raw json: %w", err)
	}

	var total float64
	for _, item := range dto.Items {
		childAmount := item.Amount * amount
		used, err := s.ingredientUsageForProduct(ctx, item.ProductID, ingredientID, childAmount, orderDate, nextPath)
		if err != nil {
			return 0, err
		}
		total += used
	}
	return total, nil
}

func (s *MonitorService) resolveIngredient(ctx context.Context, code string) (sqlc.GetIikoProductByIDRow, error) {
	code = strings.TrimSpace(code)
	product, err := s.queries.GetIikoProductByCode(ctx, code)
	if err == nil {
		return sqlc.GetIikoProductByIDRow(product), nil
	}
	if err != pgx.ErrNoRows {
		return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("get ingredient by code %s: %w", code, err)
	}
	return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("ingredient not found by code %s", code)
}
