package monitoring

import "fmt"

const DefaultMaxDepth = 12

type ProductGraph map[string]ProductRecipe

type ProductRecipe struct {
	Assembly *AssemblyRecipe
	Prepared *PreparedRecipe
}

type AssemblyRecipe struct {
	AssembledAmount float64
	Items           []RecipeItem
}

type PreparedRecipe struct {
	Items []RecipeItem
}

type RecipeItem struct {
	ProductID string
	Amount    float64
	// Code/Name/Unit описывают продукт компонента для отображения состава;
	// на расчёт расхода они не влияют.
	Code string
	Name string
	Unit string
}

type Service struct {
	maxDepth int
}

func NewService() *Service {
	return &Service{maxDepth: DefaultMaxDepth}
}

func (s *Service) CalculateIngredientUsage(graph ProductGraph, productID string, ingredientID string, amount float64) (float64, error) {
	if s == nil {
		s = NewService()
	}
	return s.calculateIngredientUsage(graph, productID, ingredientID, amount, make(map[string]bool, s.maxDepth))
}

// ComposeRecipe раскладывает продукт на компоненты его техкарты (один уровень),
// масштабируя количества на amount. Формулы те же, что и в расчёте расхода:
// сборочная карта — item.Amount * (amount / AssembledAmount),
// карта полуфабриката — item.Amount * amount.
func (s *Service) ComposeRecipe(graph ProductGraph, productID string, amount float64) []IngredientComponent {
	if productID == "" || amount == 0 {
		return nil
	}
	recipe, ok := graph[productID]
	if !ok {
		return nil
	}

	var items []RecipeItem
	scale := amount
	switch {
	case recipe.Assembly != nil:
		if recipe.Assembly.AssembledAmount > 0 {
			scale = amount / recipe.Assembly.AssembledAmount
		}
		items = recipe.Assembly.Items
	case recipe.Prepared != nil:
		items = recipe.Prepared.Items
	default:
		return nil
	}

	components := make([]IngredientComponent, 0, len(items))
	for _, item := range items {
		components = append(components, IngredientComponent{
			ProductCode: item.Code,
			ProductName: item.Name,
			Unit:        item.Unit,
			Quantity:    item.Amount * scale,
		})
	}
	return components
}

func (s *Service) calculateIngredientUsage(
	graph ProductGraph,
	productID string,
	ingredientID string,
	amount float64,
	path map[string]bool,
) (float64, error) {
	if productID == "" || amount == 0 {
		return 0, nil
	}
	if productID == ingredientID {
		return amount, nil
	}
	if path[productID] {
		return 0, nil
	}
	if len(path) >= s.maxDepth {
		return 0, fmt.Errorf("monitor calculation max depth exceeded for product %s", productID)
	}

	path[productID] = true
	defer delete(path, productID)

	recipe, ok := graph[productID]
	if !ok {
		return 0, nil
	}

	if recipe.Assembly != nil {
		return s.calculateAssemblyUsage(graph, *recipe.Assembly, ingredientID, amount, path)
	}
	if recipe.Prepared != nil {
		return s.calculatePreparedUsage(graph, *recipe.Prepared, ingredientID, amount, path)
	}
	return 0, nil
}

func (s *Service) calculateAssemblyUsage(
	graph ProductGraph,
	recipe AssemblyRecipe,
	ingredientID string,
	amount float64,
	path map[string]bool,
) (float64, error) {
	scale := amount
	if recipe.AssembledAmount > 0 {
		scale = amount / recipe.AssembledAmount
	}

	var total float64
	for _, item := range recipe.Items {
		used, err := s.calculateIngredientUsage(graph, item.ProductID, ingredientID, item.Amount*scale, path)
		if err != nil {
			return 0, err
		}
		total += used
	}
	return total, nil
}

func (s *Service) calculatePreparedUsage(
	graph ProductGraph,
	recipe PreparedRecipe,
	ingredientID string,
	amount float64,
	path map[string]bool,
) (float64, error) {
	var total float64
	for _, item := range recipe.Items {
		used, err := s.calculateIngredientUsage(graph, item.ProductID, ingredientID, item.Amount*amount, path)
		if err != nil {
			return 0, err
		}
		total += used
	}
	return total, nil
}
