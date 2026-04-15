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
	r := &JsonProductRepository{
		filePath: path,
	}
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

	var children []domain.Ingredient
	for _, c := range dto.Ingredients {
		children = append(children, c.toDomain())
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
		Quantity    float64         `json:"quantity"`
		Unit        string          `json:"unit"`
		Ingredients []ingredientDTO `json:"ingredients"`
	}

	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return err
	}

	var result []domain.Product

	for _, p := range raw {
		var ingredients []domain.Ingredient
		for _, ing := range p.Ingredients {
			ingredients = append(ingredients, ing.toDomain())
		}

		result = append(result, domain.Product{
			Name:        p.Name,
			Quantity:    p.Quantity,
			Unit:        p.Unit,
			Ingredients: ingredients,
		})
	}

	r.products = result
	return nil
}
func ingredientToDTO(i domain.Ingredient) ingredientDTO {
	dto := ingredientDTO{
		Name:     i.Name(),
		Quantity: i.Quantity(),
		Unit:     i.Unit(),
		Gross:    i.Gross(),
		Net:      i.Net(),
	}

	for _, child := range i.Ingredients() {
		dto.Ingredients = append(dto.Ingredients, ingredientToDTO(child))
	}

	return dto
}
func (r *JsonProductRepository) save() error {
	file, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var raw []interface{}

	for _, p := range r.products {
		var ingredients []ingredientDTO
		for _, ing := range p.Ingredients {
			ingredients = append(ingredients, ingredientToDTO(ing))
		}

		raw = append(raw, map[string]interface{}{
			"name":        p.Name,
			"quantity":    p.Quantity,
			"unit":        p.Unit,
			"ingredients": ingredients,
		})
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	return enc.Encode(raw)
}
func (r *JsonProductRepository) Save(p domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, prod := range r.products {
		if prod.Name == p.Name {
			r.products[i] = p
			return r.save()
		}
	}

	r.products = append(r.products, p)
	return r.save()
}

func (r *JsonProductRepository) Get(name string, quantity float64) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.products {
		if p.Name == name {
			return scaleProduct(p, quantity), nil
		}
	}

	return domain.Product{}, fmt.Errorf("продукт %s не найден", name)
}

// GetBase возвращает базовый рецепт на 1 штуку без масштабирования.
func (r *JsonProductRepository) GetBase(name string) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.products {
		if p.Name == name {
			return p, nil
		}
	}

	return domain.Product{}, fmt.Errorf("продукт %s не найден", name)
}

func (r *JsonProductRepository) List() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for _, p := range r.products {
		names = append(names, p.Name)
	}

	return names, nil
}
func scaleProduct(p domain.Product, quantity float64) domain.Product {
	factor := quantity / p.Quantity

	return domain.Product{
		Name:        p.Name,
		Quantity:    quantity,
		Unit:        p.Unit,
		Ingredients: scaleIngredients(p.Ingredients, factor),
	}
}

func scaleIngredients(ings []domain.Ingredient, factor float64) []domain.Ingredient {
	var result []domain.Ingredient

	for _, ing := range ings {
		switch v := ing.(type) {

		case domain.RawIngredient:
			result = append(result, domain.RawIngredient{
				IName:     v.IName,
				IQuantity: v.IQuantity * factor,
				IUnit:     v.IUnit,
				IGross:    v.IGross * factor,
				INet:      v.INet * factor,
			})

		case domain.SubProduct:
			result = append(result, domain.SubProduct{
				IName:        v.IName,
				IQuantity:    v.IQuantity * factor,
				IUnit:        v.IUnit,
				IGross:       v.IGross * factor,
				INet:         v.INet * factor,
				IIngredients: scaleIngredients(v.IIngredients, factor),
			})
		}
	}

	return result
}
