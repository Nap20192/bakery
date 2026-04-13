package repo

import (
	"bakery/internal/domain"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type JsonProductRepository struct {
	filePath string
	mu       sync.RWMutex
}

func NewJsonProductRepository(path string) *JsonProductRepository {
	return &JsonProductRepository{filePath: path}
}

type ingredientDTO struct {
	Type string          `json:"type"` // "raw" или "sub"
	Data json.RawMessage `json:"data"`
}

type productDTO struct {
	Name        string            `json:"name"`
	Quantity    float64           `json:"quantity"`
	Unit        string            `json:"unit"`
	Ingredients []json.RawMessage `json:"ingredients,omitempty"`
}

type rawIngredientDTO struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Gross    float64 `json:"gross"`
	Net      float64 `json:"net"`
}

type subProductDTO struct {
	rawIngredientDTO
	Ingredients []json.RawMessage `json:"ingredients,omitempty"`
}

func (r *JsonProductRepository) Save(p domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	products, err := r.loadAll()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	products[p.Name] = p
	return r.saveAll(products)
}

func (r *JsonProductRepository) Get(name string, quantity float64) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products, err := r.loadAll()
	if err != nil {
		return domain.Product{}, err
	}

	p, ok := products[name]
	if !ok {
		return domain.Product{}, fmt.Errorf("product %s not found", name)
	}

	p.Ingredients = r.scaleIngredients(p.Ingredients, quantity)
	p.Quantity = quantity
	return p, nil
}

func (r *JsonProductRepository) List() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products, err := r.loadAll()
	if err != nil {
		return nil, err
	}

	var list []string
	for name := range products {
		list = append(list, name)
	}
	return list, nil
}

func (r *JsonProductRepository) scaleIngredients(ings []domain.Ingredient, factor float64) []domain.Ingredient {
	var scaled []domain.Ingredient
	for _, ing := range ings {
		if sub, ok := ing.(domain.SubProduct); ok {
			scaled = append(scaled, domain.SubProduct{
				IName:        sub.IName,
				IQuantity:    sub.IQuantity * factor,
				IUnit:        sub.IUnit,
				IGross:       sub.IGross * factor,
				INet:         sub.INet * factor,
				IIngredients: r.scaleIngredients(sub.IIngredients, factor),
			})
		} else if raw, ok := ing.(domain.RawIngredient); ok {
			scaled = append(scaled, domain.RawIngredient{
				IName:     raw.IName,
				IQuantity: raw.IQuantity * factor,
				IUnit:     raw.IUnit,
				IGross:    raw.IGross * factor,
				INet:      raw.INet * factor,
			})
		}
	}
	return scaled
}

func (r *JsonProductRepository) loadAll() (map[string]domain.Product, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return make(map[string]domain.Product), err
	}

	var productsDTO map[string]productDTO
	if err := json.Unmarshal(data, &productsDTO); err != nil {
		return make(map[string]domain.Product), err
	}

	products := make(map[string]domain.Product, len(productsDTO))
	for key, dto := range productsDTO {
		ingredients, err := decodeIngredients(dto.Ingredients)
		if err != nil {
			return make(map[string]domain.Product), fmt.Errorf("decode ingredients for %s: %w", key, err)
		}
		products[key] = domain.Product{
			Name:        dto.Name,
			Quantity:    dto.Quantity,
			Unit:        dto.Unit,
			Ingredients: ingredients,
		}
	}

	return products, nil
}

func decodeIngredients(raw []json.RawMessage) ([]domain.Ingredient, error) {
	out := make([]domain.Ingredient, 0, len(raw))
	for _, item := range raw {
		ing, err := decodeIngredient(item)
		if err != nil {
			return nil, err
		}
		out = append(out, ing)
	}
	return out, nil
}

func decodeIngredient(raw json.RawMessage) (domain.Ingredient, error) {
	var wrapper ingredientDTO
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Type != "" {
		if len(wrapper.Data) == 0 {
			return nil, fmt.Errorf("ingredient wrapper type=%q has empty data", wrapper.Type)
		}
		switch wrapper.Type {
		case "raw":
			return decodeRawIngredient(wrapper.Data)
		case "sub":
			return decodeSubProduct(wrapper.Data)
		default:
			return nil, fmt.Errorf("unknown ingredient wrapper type %q", wrapper.Type)
		}
	}

	return decodePlainIngredient(raw)
}

func decodePlainIngredient(raw json.RawMessage) (domain.Ingredient, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("parse ingredient object: %w", err)
	}

	if _, ok := fields["ingredients"]; ok {
		return decodeSubProduct(raw)
	}
	return decodeRawIngredient(raw)
}

func decodeRawIngredient(raw json.RawMessage) (domain.Ingredient, error) {
	var dto rawIngredientDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("decode raw ingredient: %w", err)
	}
	return domain.RawIngredient{
		IName:     dto.Name,
		IQuantity: dto.Quantity,
		IUnit:     dto.Unit,
		IGross:    dto.Gross,
		INet:      dto.Net,
	}, nil
}

func decodeSubProduct(raw json.RawMessage) (domain.Ingredient, error) {
	var dto subProductDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("decode sub-product: %w", err)
	}

	ingredients, err := decodeIngredients(dto.Ingredients)
	if err != nil {
		return nil, err
	}

	return domain.SubProduct{
		IName:        dto.Name,
		IQuantity:    dto.Quantity,
		IUnit:        dto.Unit,
		IGross:       dto.Gross,
		INet:         dto.Net,
		IIngredients: ingredients,
	}, nil
}

func (r *JsonProductRepository) saveAll(products map[string]domain.Product) error {
	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}
