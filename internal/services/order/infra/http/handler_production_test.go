package orderhttp

import (
	"encoding/json"
	"testing"

	orderdomain "bakery/internal/services/order/domain"
)

func TestProductionSheetResponseCategoryID(t *testing.T) {
	t.Parallel()
	categoryID := int64(7)
	for _, id := range []*int64{&categoryID, nil} {
		response := toProductionSheetResponse(orderdomain.ProductionSheet{CategoryID: id})
		body, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		value, exists := fields["category_id"]
		if id == nil {
			if exists {
				t.Fatalf("empty category must be omitted: %s", body)
			}
		} else if string(value) != "7" {
			t.Fatalf("category_id = %s, want 7: %s", value, body)
		}
	}
}
