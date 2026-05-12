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
