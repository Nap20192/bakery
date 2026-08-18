package web

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"bakery/internal/inbound/api/contract"
)

func formatDate(value string) string {
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05.999999Z07:00"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Format("02.01.2006")
		}
	}
	return value
}

// formatDayMonth renders "21.07" for the date grids, where the weekday sits
// right above and the year never varies within a visible window.
func formatDayMonth(value string) string {
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05.999999Z07:00"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("02.01")
		}
	}
	return value
}

func formatDateTime(value string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.999999Z07:00"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Local().Format("02.01.2006 15:04")
		}
	}
	return value
}

func formatQuantity(value any) string {
	var number float64
	switch typed := value.(type) {
	case float64:
		number = typed
	case int:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case *float64:
		if typed == nil {
			return "—"
		}
		number = *typed
	default:
		return fmt.Sprint(value)
	}
	if math.Abs(number-math.Round(number)) < 0.000001 {
		return strconv.FormatInt(int64(math.Round(number)), 10)
	}
	// Dough calculations divide by assembled amounts and produce long tails such
	// as 1.3333333333333333. Three decimals is display-only rounding; the backend
	// keeps full precision. Order and production quantities never carry more than
	// one decimal, so they pass through unchanged.
	rounded := math.Round(number*1000) / 1000
	if math.Abs(rounded-math.Round(rounded)) < 0.000001 {
		return strconv.FormatInt(int64(math.Round(rounded)), 10)
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// formatQuantity3 renders ingredient weights with fixed three decimals
// (0.3 → «0.300»): the dough scale is read to the gram, so trailing zeros
// carry meaning. Order/production piece counts keep formatQuantity.
func formatQuantity3(value float64) string {
	return strconv.FormatFloat(math.Round(value*1000)/1000, 'f', 3, 64)
}

func inputQuantity(value float64) string {
	if math.Abs(value) < 0.000001 {
		return ""
	}
	return formatQuantity(value)
}

func roleName(role string) string {
	switch role {
	case "admin":
		return "Администратор"
	case "baker":
		return "Пекарь"
	case "shop":
		return "Магазин"
	default:
		return role
	}
}

func profileInitials(viewer *contract.Me) string {
	if viewer == nil {
		return "?"
	}
	source := viewer.TelegramUsername
	if source == "" {
		source = roleName(viewer.Role)
	}
	runes := []rune(strings.ToUpper(source))
	if len(runes) == 0 {
		return "?"
	}
	if len(runes) == 1 {
		return string(runes[0])
	}
	return string(runes[:2])
}

func activePath(current, target string) bool {
	if target == "/orders" {
		return current == target || (strings.HasPrefix(current, "/orders/") && current != "/orders/new" && current != "/orders/table")
	}
	return current == target || strings.HasPrefix(current, target+"/")
}

func categoryTone(value any) string {
	var color string
	switch category := value.(type) {
	case contract.Category:
		color = category.Color
	case *contract.Category:
		if category != nil {
			color = category.Color
		}
	}
	if color == "" {
		return "stone"
	}
	switch color {
	case "amber", "sky", "violet", "emerald", "rose", "stone":
		return color
	default:
		return "stone"
	}
}

func sheetTone(value any) string {
	var id int64
	switch typed := value.(type) {
	case int64:
		id = typed
	case *int64:
		if typed != nil {
			id = *typed
		}
	case int:
		id = int64(typed)
	}
	tones := []string{"sheet-olive", "sheet-copper", "sheet-blue", "sheet-plum", "sheet-moss", "sheet-clay"}
	if id < 0 {
		id = -id
	}
	return tones[id%int64(len(tones))]
}

func equalOptionalID(id int64, value *int64) bool { return value != nil && id == *value }

func commentFor(productName string, comments []contract.ItemComment) string {
	for _, item := range comments {
		if item.ProductName == productName {
			return item.Comment
		}
	}
	return ""
}

func effectiveQuantity(item contract.OrderItem) float64 {
	if item.ProducedQuantity != nil {
		return *item.ProducedQuantity
	}
	return item.ProductionQuantity
}
