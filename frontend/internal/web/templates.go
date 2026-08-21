package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"bakery/frontend/internal/application"
)

// parseTemplates builds the page template set.
//
// templates/layout.html is the shell, templates/pages/*.html hold one template
// per page.View (the name must match the View value), templates/components/*.html
// hold the partials shared between pages.
func parseTemplates() (*template.Template, error) {
	var templates *template.Template

	funcs := template.FuncMap{
		"active":           activePath,
		"add":              func(a, b int) int { return a + b },
		"canCancelOrders":  application.CanCancelOrders,
		"canManageOrders":  application.CanManageOrders,
		"canUseProduction": application.CanUseProduction,
		"categoryTone":     categoryTone,
		"commentFor":       commentFor,
		"deref":            derefID,
		"dict":             dict,
		"eqID":             equalOptionalID,
		"formatDate":       formatDate,
		"formatDateTime":   formatDateTime,
		"formatDayMonth":   formatDayMonth,
		"formatQty":        formatQuantity,
		"formatQty3":       formatQuantity3,
		"inputQty":         inputQuantity,
		"isAdmin":          application.IsAdmin,
		"isShop":           application.IsShop,
		"isset":            isset,
		"join":             strings.Join,
		"list":             func(values ...any) []any { return values },
		"pageOffset":       func(page, offset int32) int32 { return page + offset },
		"profileInitials":  profileInitials,
		"roleName":         roleName,
		"sheetTone":        sheetTone,
		"urlquery":         url.QueryEscape,
		// view renders a template chosen at run time, which {{template}} cannot do
		// because it only accepts a constant name. The layout uses it to dispatch
		// on page.View instead of listing every view in an if-chain.
		"view": func(name string, data any) (template.HTML, error) {
			var buffer bytes.Buffer
			if err := templates.ExecuteTemplate(&buffer, name, data); err != nil {
				return "", fmt.Errorf("render view %q: %w", name, err)
			}
			// The nested template was rendered by html/template, which already escaped its data.
			return template.HTML(buffer.String()), nil //nolint:gosec
		},
	}

	templates, err := template.New("layout").Funcs(funcs).ParseFS(
		frontendFiles,
		"templates/*.html",
		"templates/components/*.html",
		"templates/pages/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return templates, nil
}

// dict builds the argument map for a component, since {{template}} passes a
// single value. Keys missing from the map render as their zero value, so
// optional component parameters can simply be omitted at the call site.
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: got %d arguments, want key/value pairs", len(values))
	}
	result := make(map[string]any, len(values)/2)
	for index := 0; index < len(values); index += 2 {
		key, ok := values[index].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at position %d is not a string", index)
		}
		result[key] = values[index+1]
	}
	return result, nil
}

// isset reports whether a component was given a parameter at all, which {{if}}
// cannot tell apart from a zero value such as a count of 0.
func isset(values map[string]any, key string) bool {
	_, ok := values[key]
	return ok
}

func derefID(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
