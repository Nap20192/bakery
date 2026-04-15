package app

import (
	"fmt"

	"bakery/internal/domain"
)

type OrderService struct {
	repo domain.ProductRepository
}

func NewOrderService(repo domain.ProductRepository) *OrderService {
	return &OrderService{repo: repo}
}

type CalculateTotalResponse struct {
	Total       float64                      `json:"total"`
	Products    []CalculateTotalResponseItem `json:"products"`
	Ingredients []domain.Ingredient          `json:"ingredients,omitempty"`
}

type CalculateTotalResponseItem struct {
	Product  string `json:"product"`
	Quantity int    `json:"quantity"`
}

func (o *OrderService) CalculateTotalIngredient(order domain.Order, target string) (CalculateTotalResponse, error) {
	var grandTotal float64
	var products []CalculateTotalResponseItem

	for _, item := range order.Items {
		product, err := o.repo.Get(item.Product, float64(item.Quantity))
		if err != nil {
			return CalculateTotalResponse{}, err
		}
		countInProduct := o.countRecursive(product.Ingredients, target)
		if countInProduct > 0 {
			grandTotal += countInProduct
			products = append(products, CalculateTotalResponseItem{
				Product:  item.Product,
				Quantity: item.Quantity,
			})
		}
	}

	return CalculateTotalResponse{
		Total:    grandTotal,
		Products: products,
	}, nil
}

func (o *OrderService) countRecursive(ingredients []domain.Ingredient, target string) float64 {
	var total float64
	for _, ing := range ingredients {
		if ing.Name() == target {
			total += ing.Quantity()
		}
		total += o.countRecursive(ing.Ingredients(), target)
	}
	return total
}

// CalculateDough возвращает суммарное количество теста каждого вида по заявке.
//
// Формула: тесто_кг = количество_на_1шт_по_рецепту × заказано_штук
//
// product.Quantity в JSON — вес готового изделия (например 0.5418 кг для булочки),
// а не количество штук в рецепте. Поэтому мы берём базовый рецепт (1 шт)
// и умножаем ингредиент-тесто на количество штук в заказе.
func (o *OrderService) CalculateDough(order domain.Order, doughRepo domain.DoughRepository) ([]domain.DoughSummary, error) {
	totals := make(map[string]float64)

	for _, item := range order.Items {
		// Берём базовый рецепт на 1 штуку без масштабирования
		product, err := o.repo.GetBase(item.Product)
		if err != nil {
			return nil, fmt.Errorf("продукт %q: %w", item.Product, err)
		}
		// тесто_итого = тесто_на_1шт × кол-во_штук
		for _, ing := range product.Ingredients {
			if doughRepo.IsDough(ing.Name()) {
				totals[ing.Name()] += ing.Quantity() * float64(item.Quantity)
			}
		}
	}

	result := make([]domain.DoughSummary, 0, len(totals))
	for name, kg := range totals {
		result = append(result, domain.DoughSummary{DoughName: name, TotalKg: kg})
	}
	return result, nil
}

func (o *OrderService) CombineOrders(orders []domain.Order) domain.Order {
	combined := make(map[string]int)
	for _, order := range orders {
		for _, item := range order.Items {
			combined[item.Product] += item.Quantity
		}
	}
	var result domain.Order
	for product, quantity := range combined {
		result.Items = append(result.Items, domain.OrderItem{
			Product:  product,
			Quantity: quantity,
		})
	}
	return result
}
