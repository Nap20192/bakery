package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
	"bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/enum"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type MonitorService struct {
	queries *sqlc.Queries
	domain  *monitoringdomain.Service
}

func NewMonitorService(queries *sqlc.Queries) *MonitorService {
	return &MonitorService{
		queries: queries,
		domain:  monitoringdomain.NewService(),
	}
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
	graph := monitoringdomain.ProductGraph{}

	for _, orderItem := range order.Items {
		breakdown := monitoringdomain.IngredientDishBreakdown{
			OrderItemCode:     orderItem.Code,
			OrderItemName:     orderItem.ProductName,
			OrderItemQuantity: orderItem.Quantity,
		}

		product, err := s.queries.GetIikoProductByCode(ctx, strings.TrimSpace(orderItem.Code))
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return report, fmt.Errorf("get product by code %s: %w", orderItem.Code, err)
		}

		if err := s.LoadMonitorGraph(ctx, graph, product.ID, orderDateParam); err != nil {
			return report, fmt.Errorf("load monitor graph for product %s: %w", orderItem.Code, err)
		}

		used, err := s.domain.CalculateIngredientUsage(graph, product.ID, ingredient.ID, orderItem.Quantity)
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
	if product.Type != nil && enum.IsIikoProductType(*product.Type, enum.IikoProductTypePrepared) {
		return "кг"
	}
	return ""
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func (s *MonitorService) LoadMonitorGraph(ctx context.Context, graph monitoringdomain.ProductGraph, productID string, orderDate pgtype.Date) error {
	return s.loadMonitorGraph(ctx, graph, productID, orderDate, make(map[string]bool, monitoringdomain.DefaultMaxDepth))
}

func (s *MonitorService) loadMonitorGraph(ctx context.Context, graph monitoringdomain.ProductGraph, productID string, orderDate pgtype.Date, path map[string]bool) error {
	if productID == "" {
		return nil
	}
	if _, ok := graph[productID]; ok {
		return nil
	}
	if path[productID] || len(path) >= monitoringdomain.DefaultMaxDepth {
		return nil
	}

	assembly, err := s.queries.GetActiveAssemblyChartByProductID(ctx, sqlc.GetActiveAssemblyChartByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          orderDate,
	})
	if err == nil {
		items, err := s.queries.ListAssemblyChartItemsByChartID(ctx, assembly.ID)
		if err != nil {
			return fmt.Errorf("list assembly chart items %s: %w", assembly.ID, err)
		}

		recipeItems := make([]monitoringdomain.RecipeItem, 0, len(items))
		for _, item := range items {
			recipeItems = append(recipeItems, monitoringdomain.RecipeItem{
				ProductID: item.ProductID,
				Amount:    item.AmountIn,
			})
		}
		graph[productID] = monitoringdomain.ProductRecipe{
			Assembly: &monitoringdomain.AssemblyRecipe{
				AssembledAmount: assembly.AssembledAmount,
				Items:           recipeItems,
			},
		}

		path[productID] = true
		defer delete(path, productID)
		for _, item := range items {
			if err := s.loadMonitorGraph(ctx, graph, item.ProductID, orderDate, path); err != nil {
				return err
			}
		}
		return nil
	}
	if err != pgx.ErrNoRows {
		return fmt.Errorf("get active assembly chart: %w", err)
	}

	prepared, err := s.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          orderDate,
	})
	if err == pgx.ErrNoRows {
		graph[productID] = monitoringdomain.ProductRecipe{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get active prepared chart: %w", err)
	}

	items, err := s.queries.ListPreparedChartItemsByChartID(ctx, prepared.ID)
	if err != nil {
		return fmt.Errorf("list prepared chart items %s: %w", prepared.ID, err)
	}

	recipeItems := make([]monitoringdomain.RecipeItem, 0, len(items))
	for _, item := range items {
		recipeItems = append(recipeItems, monitoringdomain.RecipeItem{
			ProductID: item.ProductID,
			Amount:    item.Amount,
		})
	}
	graph[productID] = monitoringdomain.ProductRecipe{
		Prepared: &monitoringdomain.PreparedRecipe{Items: recipeItems},
	}

	path[productID] = true
	defer delete(path, productID)
	for _, item := range items {
		if err := s.loadMonitorGraph(ctx, graph, item.ProductID, orderDate, path); err != nil {
			return err
		}
	}
	return nil
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
