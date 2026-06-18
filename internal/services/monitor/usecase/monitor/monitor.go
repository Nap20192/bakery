package monitoruc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	monitoringdomain "bakery/internal/services/monitor/domain"
	orderdomain "bakery/internal/services/order/domain"
)

type Service struct {
	repo   Repository
	domain *monitoringdomain.Service
}

var _ UseCase = (*Service)(nil)

func NewService(repo Repository) *Service {
	return &Service{repo: repo, domain: monitoringdomain.NewService()}
}

func (s *Service) GetIngredientsByCode(ctx context.Context, code string, order orderdomain.Order) (monitoringdomain.IngredientReport, error) {
	ingredient, err := s.repo.ResolveIngredient(ctx, strings.TrimSpace(code))
	if err != nil {
		return monitoringdomain.IngredientReport{}, err
	}
	orderGraph, err := s.repo.LoadOrderGraph(ctx, order)
	if err != nil {
		return monitoringdomain.IngredientReport{}, err
	}
	return s.calculateIngredientReport(ingredient, orderGraph)
}

func (s *Service) GetBatchIngredientsByCodes(ctx context.Context, codes []string, orders []orderdomain.Order) (monitoringdomain.BatchMonitoringReport, error) {
	result := monitoringdomain.BatchMonitoringReport{
		Orders:       make([]monitoringdomain.OrderMonitoringReport, 0, len(orders)),
		TotalReports: make([]monitoringdomain.IngredientReport, 0, len(codes)),
	}
	if len(codes) == 0 || len(orders) == 0 {
		return result, nil
	}

	ingredients := make([]Ingredient, 0, len(codes))
	for _, code := range codes {
		ingredient, err := s.repo.ResolveIngredient(ctx, strings.TrimSpace(code))
		if err != nil {
			return result, err
		}
		ingredients = append(ingredients, ingredient)
	}

	totalByCode := make(map[string]monitoringdomain.IngredientReport, len(ingredients))
	breakdownByCode := make(map[string]map[string]monitoringdomain.IngredientDishBreakdown, len(ingredients))

	for _, order := range orders {
		orderGraph, err := s.repo.LoadOrderGraph(ctx, order)
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

func (s *Service) calculateIngredientReport(ingredient Ingredient, orderGraph OrderGraph) (monitoringdomain.IngredientReport, error) {
	report := monitoringdomain.IngredientReport{
		Ingredient: monitoringdomain.IngredientUsage{
			ProductCode: ingredient.Code,
			ProductName: ingredient.Name,
			Unit:        ingredient.Unit,
		},
	}
	for _, graphItem := range orderGraph.Items {
		breakdown, err := s.calculateOrderItemIngredientUsage(orderGraph.Graph, graphItem, ingredient.ProductID)
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

func (s *Service) calculateOrderItemIngredientUsage(
	graph monitoringdomain.ProductGraph,
	graphItem OrderGraphItem,
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
