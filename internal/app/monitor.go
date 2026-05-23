package app

import (
	"context"
	"errors"
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

type monitorIngredient struct {
	Code    string
	Product sqlc.GetIikoProductByIDRow
}

type orderMonitorGraph struct {
	Graph monitoringdomain.ProductGraph
	Items []orderMonitorItem
}

type orderMonitorItem struct {
	Item      orderdomain.OrderItem
	ProductID string
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
	orderGraph, err := s.loadOrderMonitorGraph(ctx, order)
	if err != nil {
		return report, err
	}
	return s.calculateIngredientReport(monitorIngredient{Code: strings.TrimSpace(code), Product: ingredient}, orderGraph)
}

func (s *MonitorService) resolveIngredients(ctx context.Context, codes []string) ([]monitorIngredient, error) {
	ingredients := make([]monitorIngredient, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		ingredient, err := s.resolveIngredient(ctx, code)
		if err != nil {
			return nil, err
		}
		ingredients = append(ingredients, monitorIngredient{
			Code:    code,
			Product: ingredient,
		})
	}
	return ingredients, nil
}

func (s *MonitorService) GetBatchIngredientsByCodes(ctx context.Context, codes []string, orders []orderdomain.Order) (monitoringdomain.BatchMonitoringReport, error) {
	result := monitoringdomain.BatchMonitoringReport{
		Orders:       make([]monitoringdomain.OrderMonitoringReport, 0, len(orders)),
		TotalReports: make([]monitoringdomain.IngredientReport, 0, len(codes)),
	}
	if s == nil || s.queries == nil {
		return result, fmt.Errorf("monitor service is not initialized")
	}
	if len(codes) == 0 || len(orders) == 0 {
		return result, nil
	}

	ingredients, err := s.resolveIngredients(ctx, codes)
	if err != nil {
		return result, err
	}

	totalByCode := make(map[string]monitoringdomain.IngredientReport, len(ingredients))
	breakdownByCode := make(map[string]map[string]monitoringdomain.IngredientDishBreakdown, len(ingredients))

	for _, order := range orders {
		orderGraph, err := s.loadOrderMonitorGraph(ctx, order)
		if err != nil {
			return result, err
		}
		orderReport := monitoringdomain.OrderMonitoringReport{
			OrderNumber: order.Number,
			Reports:     make([]monitoringdomain.IngredientReport, 0, len(ingredients)),
		}
		for _, ingredient := range ingredients {
			report, err := s.calculateIngredientReport(ingredient, orderGraph)
			if err != nil {
				return result, err
			}
			orderReport.Reports = append(orderReport.Reports, report)

			total := totalByCode[ingredient.Code]
			if total.Ingredient.ProductCode == "" {
				total.Ingredient = report.Ingredient
				total.Ingredient.Quantity = 0
			}
			total.Ingredient.Quantity += report.Ingredient.Quantity
			totalByCode[ingredient.Code] = total

			if _, ok := breakdownByCode[ingredient.Code]; !ok {
				breakdownByCode[ingredient.Code] = make(map[string]monitoringdomain.IngredientDishBreakdown)
			}
			for _, breakdown := range report.Breakdown {
				key := breakdown.OrderItemCode + "\x00" + breakdown.OrderItemName
				existing := breakdownByCode[ingredient.Code][key]
				if existing.OrderItemCode == "" {
					existing = monitoringdomain.IngredientDishBreakdown{
						OrderItemCode: breakdown.OrderItemCode,
						OrderItemName: breakdown.OrderItemName,
					}
				}
				existing.OrderItemQuantity += breakdown.OrderItemQuantity
				existing.IngredientQuantity += breakdown.IngredientQuantity
				breakdownByCode[ingredient.Code][key] = existing
			}
		}
		result.Orders = append(result.Orders, orderReport)
	}

	for _, ingredient := range ingredients {
		total, ok := totalByCode[ingredient.Code]
		if !ok {
			continue
		}
		for _, breakdown := range breakdownByCode[ingredient.Code] {
			if breakdown.IngredientQuantity > 0 {
				total.Breakdown = append(total.Breakdown, breakdown)
			}
		}
		sort.Slice(total.Breakdown, func(i, j int) bool {
			return total.Breakdown[i].IngredientQuantity > total.Breakdown[j].IngredientQuantity
		})
		result.TotalReports = append(result.TotalReports, total)
	}

	return result, nil
}

func (s *MonitorService) loadOrderMonitorGraph(ctx context.Context, order orderdomain.Order) (orderMonitorGraph, error) {
	orderDateParam := pgDate(monitorOrderDate(order))
	result := orderMonitorGraph{
		Graph: monitoringdomain.ProductGraph{},
		Items: make([]orderMonitorItem, 0, len(order.Items)),
	}
	productsByCode := make(map[string]sqlc.GetIikoProductByCodeRow, len(order.Items))
	for _, item := range order.Items {
		code := strings.TrimSpace(item.Code)
		product, ok := productsByCode[code]
		if !ok {
			var err error
			product, err = s.queries.GetIikoProductByCode(ctx, code)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				return result, fmt.Errorf("get product by code %s: %w", item.Code, err)
			}
			productsByCode[code] = product
		}
		if err := s.LoadMonitorGraph(ctx, result.Graph, product.ID, orderDateParam); err != nil {
			return result, fmt.Errorf("load monitor graph for product %s: %w", item.Code, err)
		}
		result.Items = append(result.Items, orderMonitorItem{
			Item:      item,
			ProductID: product.ID,
		})
	}
	return result, nil
}

func monitorOrderDate(order orderdomain.Order) time.Time {
	orderDate := order.CreatedAt
	if !order.FulfillmentDate.IsZero() {
		orderDate = order.FulfillmentDate
	}
	if orderDate.IsZero() {
		orderDate = time.Now().UTC()
	}
	return orderDate
}

func (s *MonitorService) calculateIngredientReport(
	ingredient monitorIngredient,
	orderGraph orderMonitorGraph,
) (monitoringdomain.IngredientReport, error) {
	report := monitoringdomain.IngredientReport{
		Ingredient: monitoringdomain.IngredientUsage{
			ProductCode: ingredient.Product.Code,
			ProductName: ingredient.Product.Name,
			Unit:        monitorUnit(ingredient.Product),
		},
	}

	for _, graphItem := range orderGraph.Items {
		breakdown, err := s.calculateOrderItemIngredientUsage(orderGraph.Graph, graphItem, ingredient.Product.ID)
		if err != nil {
			return report, err
		}
		if breakdown.IngredientQuantity <= 0 {
			continue
		}
		report.Ingredient.Quantity += breakdown.IngredientQuantity
		report.Breakdown = append(report.Breakdown, breakdown)
	}

	sort.Slice(report.Breakdown, func(i, j int) bool {
		return report.Breakdown[i].IngredientQuantity > report.Breakdown[j].IngredientQuantity
	})
	return report, nil
}

func (s *MonitorService) calculateOrderItemIngredientUsage(
	graph monitoringdomain.ProductGraph,
	graphItem orderMonitorItem,
	ingredientID string,
) (monitoringdomain.IngredientDishBreakdown, error) {
	orderItem := graphItem.Item
	productionQuantity := orderItem.ProductionQuantity()
	breakdown := monitoringdomain.IngredientDishBreakdown{
		OrderItemCode:     orderItem.Code,
		OrderItemName:     orderItem.ProductName,
		OrderItemQuantity: productionQuantity,
	}

	used, err := s.domain.CalculateIngredientUsage(graph, graphItem.ProductID, ingredientID, productionQuantity)
	if err != nil {
		return breakdown, fmt.Errorf("calculate ingredient usage for product %s: %w", orderItem.Code, err)
	}
	breakdown.IngredientQuantity = used
	return breakdown, nil
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
	if path[productID] {
		return fmt.Errorf("monitor graph cycle detected at product %s", productID)
	}
	if _, ok := graph[productID]; ok {
		return nil
	}
	if len(path) >= monitoringdomain.DefaultMaxDepth {
		return fmt.Errorf("monitor graph max depth exceeded at product %s", productID)
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
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get active assembly chart: %w", err)
	}

	prepared, err := s.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          orderDate,
	})
	if errors.Is(err, pgx.ErrNoRows) {
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
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("get ingredient by code %s: %w", code, err)
	}
	return sqlc.GetIikoProductByIDRow{}, fmt.Errorf("ingredient not found by code %s", code)
}
