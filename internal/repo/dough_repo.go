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

	for _, d := range raw {
		var ingredients []domain.Ingredient
		for _, ing := range d.Ingredients {
			ingredients = append(ingredients, ing.toDomain())
		}
		r.doughs = append(r.doughs, domain.Dough{
			Name:        d.Name,
			Quantity:    d.Quantity,
			Unit:        d.Unit,
			Ingredients: ingredients,
		})
		r.index[d.Name] = struct{}{}
	}

	return nil
}

func (r *JsonDoughRepository) Get(name string) (domain.Dough, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, d := range r.doughs {
		if d.Name == name {
			return d, nil
		}
	}
	return domain.Dough{}, fmt.Errorf("тесто %q: %w", name, domain.ErrNotFound)
}

func (r *JsonDoughRepository) List() ([]domain.Dough, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Dough, len(r.doughs))
	copy(result, r.doughs)
	return result, nil
}

func (r *JsonDoughRepository) IsDough(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.index[name]
	return ok
}
