// Package monitoruc is the application layer of the monitoring service: it
// computes dough/ingredient usage for orders. Data access (ingredient lookup,
// recipe-graph loading) is behind the Repository port; calculation uses the
// monitoring domain.
package monitoruc

import (
	"context"

	monitoringdomain "bakery/internal/services/monitor/domain"
	orderdomain "bakery/internal/services/order/domain"
)

type UseCase interface {
	GetIngredientsByCode(ctx context.Context, code string, order orderdomain.Order) (monitoringdomain.IngredientReport, error)
	GetBatchIngredientsByCodes(ctx context.Context, codes []string, orders []orderdomain.Order) (monitoringdomain.BatchMonitoringReport, error)
}

// Repository loads the data the calculation needs from the iiko snapshot.
type Repository interface {
	ResolveIngredient(ctx context.Context, code string) (Ingredient, error)
	LoadOrderGraph(ctx context.Context, order orderdomain.Order) (OrderGraph, error)
}

// Ingredient is the resolved monitored product (transport-agnostic).
type Ingredient struct {
	ProductID string
	Code      string
	Name      string
	Unit      string
}

// OrderGraph is an order's recipe graph plus its resolved items.
type OrderGraph struct {
	Graph monitoringdomain.ProductGraph
	Items []OrderGraphItem
}

type OrderGraphItem struct {
	Item      orderdomain.OrderItem
	ProductID string
}
