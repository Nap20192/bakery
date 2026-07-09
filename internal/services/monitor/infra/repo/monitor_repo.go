// Package monitorrepo is the persistence adapter of the monitoring service. It
// resolves monitored ingredients and loads recipe graphs from the iiko snapshot
// (assembly/prepared charts) via sqlc.
package monitorrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/enum"
	monitoringdomain "bakery/internal/services/monitor/domain"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
	orderdomain "bakery/internal/services/order/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type MonitorRepository struct {
	queries *sqlc.Queries
}

var _ monitoruc.Repository = (*MonitorRepository)(nil)

func New(queries *sqlc.Queries) *MonitorRepository {
	return &MonitorRepository{queries: queries}
}

func (r *MonitorRepository) ResolveIngredient(ctx context.Context, code string) (monitoruc.Ingredient, error) {
	code = strings.TrimSpace(code)
	product, err := r.queries.GetIikoProductByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return monitoruc.Ingredient{}, fmt.Errorf("ingredient not found by code %s", code)
		}
		return monitoruc.Ingredient{}, fmt.Errorf("get ingredient by code %s: %w", code, err)
	}
	unit := strings.TrimSpace(product.MeasureUnit)
	if unit == "" && product.Type != nil && enum.IsIikoProductType(*product.Type, enum.IikoProductTypePrepared) {
		unit = "кг"
	}
	return monitoruc.Ingredient{
		ProductID: product.ID,
		Code:      product.Code,
		Name:      product.Name,
		Unit:      unit,
	}, nil
}

func (r *MonitorRepository) LoadOrderGraph(ctx context.Context, order orderdomain.Order) (monitoruc.OrderGraph, error) {
	orderDateParam := pgDate(monitorOrderDate(order))
	result := monitoruc.OrderGraph{
		Graph: monitoringdomain.ProductGraph{},
		Items: make([]monitoruc.OrderGraphItem, 0, len(order.Items)),
	}
	productsByCode := make(map[string]sqlc.GetIikoProductByCodeRow, len(order.Items))
	for _, item := range order.Items {
		code := strings.TrimSpace(item.Code)
		product, ok := productsByCode[code]
		if !ok {
			var err error
			product, err = r.queries.GetIikoProductByCode(ctx, code)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				return result, fmt.Errorf("get product by code %s: %w", item.Code, err)
			}
			productsByCode[code] = product
		}
		if err := r.loadGraph(ctx, result.Graph, product.ID, orderDateParam, make(map[string]bool, monitoringdomain.DefaultMaxDepth)); err != nil {
			return result, fmt.Errorf("load monitor graph for product %s: %w", item.Code, err)
		}
		result.Items = append(result.Items, monitoruc.OrderGraphItem{Item: item, ProductID: product.ID})
	}
	return result, nil
}

func (r *MonitorRepository) loadGraph(ctx context.Context, graph monitoringdomain.ProductGraph, productID string, orderDate pgtype.Date, path map[string]bool) error {
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

	assembly, err := r.queries.GetActiveAssemblyChartByProductID(ctx, sqlc.GetActiveAssemblyChartByProductIDParams{
		AssembledProductID: productID,
		OrderDate:          orderDate,
	})
	if err == nil {
		items, err := r.queries.ListAssemblyChartItemsByChartID(ctx, assembly.ID)
		if err != nil {
			return fmt.Errorf("list assembly chart items %s: %w", assembly.ID, err)
		}
		recipeItems := make([]monitoringdomain.RecipeItem, 0, len(items))
		for _, item := range items {
			recipeItems = append(recipeItems, monitoringdomain.RecipeItem{
				ProductID: item.ProductID,
				Amount:    item.AmountIn,
				Code:      item.ProductCode,
				Name:      item.ProductName,
				Unit:      item.MeasureUnit,
			})
		}
		graph[productID] = monitoringdomain.ProductRecipe{
			Assembly: &monitoringdomain.AssemblyRecipe{AssembledAmount: assembly.AssembledAmount, Items: recipeItems},
		}
		path[productID] = true
		defer delete(path, productID)
		for _, item := range items {
			if err := r.loadGraph(ctx, graph, item.ProductID, orderDate, path); err != nil {
				return err
			}
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get active assembly chart: %w", err)
	}

	prepared, err := r.queries.GetActivePreparedChartFullByProductID(ctx, sqlc.GetActivePreparedChartFullByProductIDParams{
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

	items, err := r.queries.ListPreparedChartItemsByChartID(ctx, prepared.ID)
	if err != nil {
		return fmt.Errorf("list prepared chart items %s: %w", prepared.ID, err)
	}
	recipeItems := make([]monitoringdomain.RecipeItem, 0, len(items))
	for _, item := range items {
		recipeItems = append(recipeItems, monitoringdomain.RecipeItem{
			ProductID: item.ProductID,
			Amount:    item.Amount,
			Code:      item.ProductCode,
			Name:      item.ProductName,
			Unit:      item.MeasureUnit,
		})
	}
	graph[productID] = monitoringdomain.ProductRecipe{Prepared: &monitoringdomain.PreparedRecipe{Items: recipeItems}}
	path[productID] = true
	defer delete(path, productID)
	for _, item := range items {
		if err := r.loadGraph(ctx, graph, item.ProductID, orderDate, path); err != nil {
			return err
		}
	}
	return nil
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

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}
