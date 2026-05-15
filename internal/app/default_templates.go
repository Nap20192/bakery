package app

import (
	"strings"

	orderdomain "bakery/internal/domain/order"
)

func parseDefaultOrderTemplates(raw string) []orderdomain.OrderTemplate {
	var templates []orderdomain.OrderTemplate
	var currentName string
	var currentLines []string

	flush := func() {
		if currentName == "" || len(currentLines) == 0 {
			return
		}
		bodyLines := make([]string, 0, len(currentLines)+1)
		bodyLines = append(bodyLines, currentName)
		bodyLines = append(bodyLines, currentLines...)
		templates = append(templates, orderdomain.OrderTemplate{
			Name: currentName,
			Body: strings.Join(bodyLines, "\n"),
		})
	}

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if orderdomain.IsTemplateHeaderLine(line) {
			flush()
			currentName = line
			currentLines = currentLines[:0]
			continue
		}
		if currentName == "" {
			continue
		}
		currentLines = append(currentLines, line)
	}
	flush()

	return templates
}

func normalizeTemplateName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
