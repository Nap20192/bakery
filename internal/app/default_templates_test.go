package app

import "testing"

func TestParseDefaultDishCatalogItems(t *testing.T) {
	items := parseDefaultDishCatalogItems(`
КОКРОКИ
15542 Кокрок с картофелем 0
15544 Кокрок с творогом 0

САМСА И УЧПУЧМАК
15646 Самса с курицей 0
broken line
`)
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3: %#v", len(items), items)
	}
	if items[0].Code != "15542" || items[0].Name != "Кокрок с картофелем" || items[0].Theme != "КОКРОКИ" || items[0].SortOrder != 1 {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[2].Code != "15646" || items[2].Name != "Самса с курицей" || items[2].Theme != "САМСА И УЧПУЧМАК" || items[2].SortOrder != 3 {
		t.Fatalf("third item = %#v", items[2])
	}
}
