package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"bakery/internal/domain"
)

type JsonProductRepository struct {
	filePath string
	mu       sync.RWMutex
	products []domain.Product
}

func NewJsonProductRepository(path string) domain.ProductRepository {
	r := &JsonProductRepository{filePath: path}
	_ = r.load()
	return r
}

type ingredientDTO struct {
	Name        string          `json:"name"`
	Quantity    float64         `json:"quantity"`
	Unit        string          `json:"unit"`
	Gross       float64         `json:"gross"`
	Net         float64         `json:"net"`
	Ingredients []ingredientDTO `json:"ingredients,omitempty"`
}

func (dto ingredientDTO) toDomain() domain.Ingredient {
	if len(dto.Ingredients) == 0 {
		return domain.RawIngredient{
			IName:     dto.Name,
			IQuantity: dto.Quantity,
			IUnit:     dto.Unit,
			IGross:    dto.Gross,
			INet:      dto.Net,
		}
	}

	children := make([]domain.Ingredient, 0, len(dto.Ingredients))
	for _, child := range dto.Ingredients {
		children = append(children, child.toDomain())
	}

	return domain.SubProduct{
		IName:        dto.Name,
		IQuantity:    dto.Quantity,
		IUnit:        dto.Unit,
		IGross:       dto.Gross,
		INet:         dto.Net,
		IIngredients: children,
	}
}

func (r *JsonProductRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.products = []domain.Product{}
			return nil
		}
		return err
	}
	defer file.Close()

	var raw []struct {
		Name        string          `json:"name"`
		Type        string          `json:"type"`
		Quantity    float64         `json:"quantity"`
		Unit        string          `json:"unit"`
		Ingredients []ingredientDTO `json:"ingredients"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return err
	}

	products := make([]domain.Product, 0, len(raw))
	for _, product := range raw {
		ingredients := make([]domain.Ingredient, 0, len(product.Ingredients))
		for _, ingredient := range product.Ingredients {
			ingredients = append(ingredients, ingredient.toDomain())
		}
		products = append(products, domain.Product{
			Name:        product.Name,
			Type:        product.Type,
			Quantity:    product.Quantity,
			Unit:        product.Unit,
			Ingredients: ingredients,
		})
	}

	r.products = products
	return nil
}

func (r *JsonProductRepository) Save(product domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.products {
		if existing.Name == product.Name {
			r.products[i] = product
			return nil
		}
	}
	r.products = append(r.products, product)
	return nil
}

func (r *JsonProductRepository) Get(name string, quantity float64) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, product := range r.products {
		if product.Name == name {
			return scaleProduct(product, quantity), nil
		}
	}
	return domain.Product{}, fmt.Errorf("продукт %s: %w", name, domain.ErrNotFound)
}

func (r *JsonProductRepository) GetBase(name string) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, product := range r.products {
		if product.Name == name {
			return product, nil
		}
	}
	return domain.Product{}, fmt.Errorf("продукт %s: %w", name, domain.ErrNotFound)
}

func (r *JsonProductRepository) List() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.products))
	for _, product := range r.products {
		names = append(names, product.Name)
	}
	return names, nil
}

func scaleProduct(product domain.Product, quantity float64) domain.Product {
	if product.Quantity == 0 {
		return product
	}
	factor := quantity / product.Quantity
	return domain.Product{
		Name:        product.Name,
		Type:        product.Type,
		Quantity:    quantity,
		Unit:        product.Unit,
		Ingredients: scaleIngredients(product.Ingredients, factor),
	}
}

func scaleIngredients(ingredients []domain.Ingredient, factor float64) []domain.Ingredient {
	scaled := make([]domain.Ingredient, 0, len(ingredients))
	for _, ingredient := range ingredients {
		switch value := ingredient.(type) {
		case domain.RawIngredient:
			scaled = append(scaled, domain.RawIngredient{
				IName:     value.IName,
				IQuantity: value.IQuantity * factor,
				IUnit:     value.IUnit,
				IGross:    value.IGross * factor,
				INet:      value.INet * factor,
			})
		case domain.SubProduct:
			scaled = append(scaled, domain.SubProduct{
				IName:        value.IName,
				IQuantity:    value.IQuantity * factor,
				IUnit:        value.IUnit,
				IGross:       value.IGross * factor,
				INet:         value.INet * factor,
				IIngredients: scaleIngredients(value.IIngredients, factor),
			})
		}
	}
	return scaled
}
