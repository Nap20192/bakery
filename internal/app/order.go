package app

import (
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
