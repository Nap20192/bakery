package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"bakery/internal/domain"
)

type JsonDoughRepository struct {
	filePath string
	mu       sync.RWMutex
	doughs   []domain.Dough
	index    map[string]struct{}
}

func NewJsonDoughRepository(path string) (domain.DoughRepository, error) {
	r := &JsonDoughRepository{
		filePath: path,
		index:    make(map[string]struct{}),
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *JsonDoughRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.doughs = []domain.Dough{}
			r.index = make(map[string]struct{})
			return nil
		}
		return fmt.Errorf("dough repo: open %s: %w", r.filePath, err)
	}
	defer file.Close()

	var raw []struct {
		Name        string          `json:"name"`
		Quantity    float64         `json:"quantity"`
		Unit        string          `json:"unit"`
		Ingredients []ingredientDTO `json:"ingredients"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return fmt.Errorf("dough repo: decode: %w", err)
	}

	r.doughs = make([]domain.Dough, 0, len(raw))
	r.index = make(map[string]struct{}, len(raw))
	for _, dough := range raw {
		ingredients := make([]domain.Ingredient, 0, len(dough.Ingredients))
		for _, ingredient := range dough.Ingredients {
			ingredients = append(ingredients, ingredient.toDomain())
		}
		r.doughs = append(r.doughs, domain.Dough{
			Name:        dough.Name,
			Quantity:    dough.Quantity,
			Unit:        dough.Unit,
			Ingredients: ingredients,
		})
		r.index[dough.Name] = struct{}{}
	}
	return nil
}

func (r *JsonDoughRepository) Get(name string) (domain.Dough, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, dough := range r.doughs {
		if dough.Name == name {
			return dough, nil
		}
	}
	return domain.Dough{}, fmt.Errorf("тесто %q: %w", name, domain.ErrNotFound)
}

func (r *JsonDoughRepository) List() ([]domain.Dough, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doughs := make([]domain.Dough, len(r.doughs))
	copy(doughs, r.doughs)
	return doughs, nil
}

func (r *JsonDoughRepository) IsDough(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.index[name]
	return ok
}
