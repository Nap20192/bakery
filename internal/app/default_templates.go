package app

import (
	"strings"

	orderdomain "bakery/internal/domain/order"
)

func normalizeTemplateName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

type defaultDishCatalogItem struct {
	Code  string
	Name  string
	Theme string
}

func parseDefaultDishCatalogItems(raw string) []defaultDishCatalogItem {
	var items []defaultDishCatalogItem
	currentTheme := ""
	spec := orderdomain.NewOrderSpec()

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if orderdomain.IsTemplateHeaderLine(line) {
			currentTheme = line
			continue
		}
		if currentTheme == "" {
			continue
		}
		parsed := orderdomain.ParseOrderLine(orderdomain.BulkOrderLine{Raw: line})
		if parsed.Code == "" || parsed.Name == "" || !spec.Quantity.IsValid(parsed) {
			continue
		}
		items = append(items, defaultDishCatalogItem{
			Code:  parsed.Code,
			Name:  parsed.Name,
			Theme: currentTheme,
		})
	}

	return items
}
