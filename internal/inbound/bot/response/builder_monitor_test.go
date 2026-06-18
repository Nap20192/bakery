package response

import (
	"strings"
	"testing"

	monitoringdomain "bakery/internal/services/monitor/domain"
	orderdomain "bakery/internal/services/order/domain"
)

func TestMonitorReportsHidesProductCodes(t *testing.T) {
	report := monitoringdomain.IngredientReport{
		Ingredient: monitoringdomain.IngredientUsage{
			ProductCode: "12345",
			ProductName: "Тесто дрожжевое",
			Unit:        "кг",
			Quantity:    12,
		},
		Breakdown: []monitoringdomain.IngredientDishBreakdown{
			{
				OrderItemCode:      "67890",
				OrderItemName:      "Сосиска в тесте",
				OrderItemQuantity:  5,
				IngredientQuantity: 2,
			},
		},
	}

	got := responses.MonitorReports(orderdomain.Order{Number: "Г.25.05.26.001"}, []monitoringdomain.IngredientReport{report})

	for _, hidden := range []string{"12345", "67890"} {
		if strings.Contains(got, hidden) {
			t.Fatalf("MonitorReports() = %q, should not contain product code %q", got, hidden)
		}
	}
	for _, visible := range []string{"Тесто дрожжевое", "Сосиска в тесте"} {
		if !strings.Contains(got, visible) {
			t.Fatalf("MonitorReports() = %q, want to contain %q", got, visible)
		}
	}
}

func TestBatchMonitorReportsHidesProductCodes(t *testing.T) {
	report := monitoringdomain.BatchMonitoringReport{
		Orders: []monitoringdomain.OrderMonitoringReport{
			{
				OrderNumber: "Г.25.05.26.001",
				Reports: []monitoringdomain.IngredientReport{
					monitorReportFixture("11111", "22222"),
				},
			},
		},
		TotalReports: []monitoringdomain.IngredientReport{
			monitorReportFixture("33333", "44444"),
		},
	}

	got := responses.BatchMonitorReports(report)

	for _, hidden := range []string{"11111", "22222", "33333", "44444"} {
		if strings.Contains(got, hidden) {
			t.Fatalf("BatchMonitorReports() = %q, should not contain product code %q", got, hidden)
		}
	}
	for _, visible := range []string{"Тесто дрожжевое", "Сосиска в тесте"} {
		if !strings.Contains(got, visible) {
			t.Fatalf("BatchMonitorReports() = %q, want to contain %q", got, visible)
		}
	}
}

func monitorReportFixture(ingredientCode string, orderItemCode string) monitoringdomain.IngredientReport {
	return monitoringdomain.IngredientReport{
		Ingredient: monitoringdomain.IngredientUsage{
			ProductCode: ingredientCode,
			ProductName: "Тесто дрожжевое",
			Unit:        "кг",
			Quantity:    12,
		},
		Breakdown: []monitoringdomain.IngredientDishBreakdown{
			{
				OrderItemCode:      orderItemCode,
				OrderItemName:      "Сосиска в тесте",
				OrderItemQuantity:  5,
				IngredientQuantity: 2,
			},
		},
	}
}
