package monitoring

import "testing"

func TestService_CalculateIngredientUsage_AssemblyScale(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dish": {
			Assembly: &AssemblyRecipe{
				AssembledAmount: 10,
				Items: []RecipeItem{
					{ProductID: "dough", Amount: 2},
				},
			},
		},
	}

	got, err := svc.CalculateIngredientUsage(graph, "dish", "dough", 5)
	if err != nil {
		t.Fatalf("CalculateIngredientUsage returned error: %v", err)
	}
	if want := 1.0; !floatEqual(got, want) {
		t.Fatalf("usage = %v, want %v", got, want)
	}
}

func TestService_CalculateIngredientUsage_PreparedAmount(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dish": {
			Prepared: &PreparedRecipe{
				Items: []RecipeItem{
					{ProductID: "dough", Amount: 0.25},
				},
			},
		},
	}

	got, err := svc.CalculateIngredientUsage(graph, "dish", "dough", 8)
	if err != nil {
		t.Fatalf("CalculateIngredientUsage returned error: %v", err)
	}
	if want := 2.0; !floatEqual(got, want) {
		t.Fatalf("usage = %v, want %v", got, want)
	}
}

func TestService_CalculateIngredientUsage_NestedRecipe(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dish": {
			Assembly: &AssemblyRecipe{
				AssembledAmount: 2,
				Items: []RecipeItem{
					{ProductID: "semi", Amount: 1},
				},
			},
		},
		"semi": {
			Prepared: &PreparedRecipe{
				Items: []RecipeItem{
					{ProductID: "dough", Amount: 0.4},
				},
			},
		},
	}

	got, err := svc.CalculateIngredientUsage(graph, "dish", "dough", 6)
	if err != nil {
		t.Fatalf("CalculateIngredientUsage returned error: %v", err)
	}
	if want := 1.2; !floatEqual(got, want) {
		t.Fatalf("usage = %v, want %v", got, want)
	}
}

func TestService_ComposeRecipe_PreparedScalesByAmount(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dough": {
			Prepared: &PreparedRecipe{
				Items: []RecipeItem{
					{ProductID: "flour", Amount: 0.6, Code: "100", Name: "Мука", Unit: "кг"},
					{ProductID: "water", Amount: 0.3, Code: "101", Name: "Вода", Unit: "л"},
				},
			},
		},
	}

	got := svc.ComposeRecipe(graph, "dough", 5)

	if len(got) != 2 {
		t.Fatalf("ComposeRecipe returned %d components, want 2", len(got))
	}
	if got[0].ProductName != "Мука" || !floatEqual(got[0].Quantity, 3.0) || got[0].Unit != "кг" {
		t.Fatalf("component[0] = %+v, want Мука 3 кг", got[0])
	}
	if got[1].ProductName != "Вода" || !floatEqual(got[1].Quantity, 1.5) {
		t.Fatalf("component[1] = %+v, want Вода 1.5", got[1])
	}
}

func TestService_ComposeRecipe_AssemblyScalesByAssembledAmount(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dough": {
			Assembly: &AssemblyRecipe{
				AssembledAmount: 10,
				Items: []RecipeItem{
					{ProductID: "flour", Amount: 6, Code: "100", Name: "Мука", Unit: "кг"},
				},
			},
		},
	}

	got := svc.ComposeRecipe(graph, "dough", 5)

	if len(got) != 1 {
		t.Fatalf("ComposeRecipe returned %d components, want 1", len(got))
	}
	if !floatEqual(got[0].Quantity, 3.0) {
		t.Fatalf("component quantity = %v, want 3", got[0].Quantity)
	}
}

func TestService_ComposeRecipe_UnknownOrZeroAmountReturnsNil(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dough": {Prepared: &PreparedRecipe{Items: []RecipeItem{{ProductID: "flour", Amount: 1}}}},
	}

	if got := svc.ComposeRecipe(graph, "missing", 5); got != nil {
		t.Fatalf("ComposeRecipe(missing) = %v, want nil", got)
	}
	if got := svc.ComposeRecipe(graph, "dough", 0); got != nil {
		t.Fatalf("ComposeRecipe(zero amount) = %v, want nil", got)
	}
}

func floatEqual(a, b float64) bool {
	const epsilon = 0.000000001
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func TestService_CalculateIngredientUsage_CycleReturnsZero(t *testing.T) {
	svc := NewService()
	graph := ProductGraph{
		"dish": {
			Prepared: &PreparedRecipe{
				Items: []RecipeItem{
					{ProductID: "semi", Amount: 1},
				},
			},
		},
		"semi": {
			Prepared: &PreparedRecipe{
				Items: []RecipeItem{
					{ProductID: "dish", Amount: 1},
				},
			},
		},
	}

	got, err := svc.CalculateIngredientUsage(graph, "dish", "dough", 1)
	if err != nil {
		t.Fatalf("CalculateIngredientUsage returned error: %v", err)
	}
	if got != 0 {
		t.Fatalf("usage = %v, want 0", got)
	}
}
