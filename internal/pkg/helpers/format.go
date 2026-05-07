package helpers

import (
	"fmt"
	"strings"
)

func FormatQuantity(quantity float64) string {
	result := fmt.Sprintf("%.3f", quantity)
	result = strings.TrimRight(result, "0")
	result = strings.TrimRight(result, ".")
	if result == "" {
		return "0"
	}
	return result
}
