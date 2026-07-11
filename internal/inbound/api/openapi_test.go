package api

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestRoutesMatchOpenAPISpec держит docs/api/openapi.yaml и реальные роуты в
// синхроне: каждый зарегистрированный в mux путь обязан быть описан в спеке,
// и каждая операция спеки — зарегистрирована в коде. Роуты собираются сканом
// исходников (mux.Handle/HandleFunc с паттерном "METHOD /path"), спека —
// построчным разбором фиксированного формата файла (пути с отступом в два
// пробела под paths:, методы — в четыре).
func TestRoutesMatchOpenAPISpec(t *testing.T) {
	root := repoRoot(t)

	registered := scanRegisteredRoutes(t, root)
	specified := scanSpecOperations(t, filepath.Join(root, "docs", "api", "openapi.yaml"))

	var missingInSpec, missingInCode []string
	for route := range registered {
		if _, ok := specified[route]; !ok {
			missingInSpec = append(missingInSpec, route)
		}
	}
	for route := range specified {
		if _, ok := registered[route]; !ok {
			missingInCode = append(missingInCode, route)
		}
	}
	sort.Strings(missingInSpec)
	sort.Strings(missingInCode)

	if len(missingInSpec) > 0 {
		t.Errorf("routes registered in code but absent from docs/api/openapi.yaml:\n  %s", strings.Join(missingInSpec, "\n  "))
	}
	if len(missingInCode) > 0 {
		t.Errorf("operations in docs/api/openapi.yaml with no registered route:\n  %s", strings.Join(missingInCode, "\n  "))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// Тест живёт в internal/inbound/api — корень на три уровня выше.
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root not found at %s: %v", root, err)
	}
	return root
}

var routePattern = regexp.MustCompile(`mux\.Handle(?:Func)?\(\s*"([A-Z]+) ([^"]+)"`)

// scanRegisteredRoutes collects "METHOD /path" patterns from every delivery
// adapter (services' infra/http and this package).
func scanRegisteredRoutes(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	routes := make(map[string]struct{})
	for _, dir := range []string{
		filepath.Join(root, "internal", "services"),
		filepath.Join(root, "internal", "inbound", "api"),
	} {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path) //nolint:gosec // тест читает исходники собственного репозитория
			if err != nil {
				return err
			}
			for _, match := range routePattern.FindAllStringSubmatch(string(content), -1) {
				routes[match[1]+" "+match[2]] = struct{}{}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(routes) == 0 {
		t.Fatal("no registered routes found — did the registration pattern change?")
	}
	return routes
}

var specMethods = map[string]string{
	"get": "GET", "post": "POST", "put": "PUT", "patch": "PATCH", "delete": "DELETE",
}

// scanSpecOperations reads "METHOD /path" pairs from the OpenAPI file. It
// relies on the file's fixed layout: path keys indented two spaces under
// paths:, method keys four.
func scanSpecOperations(t *testing.T, specPath string) map[string]struct{} {
	t.Helper()
	file, err := os.Open(specPath) //nolint:gosec // фиксированный путь docs/api/openapi.yaml внутри репозитория
	if err != nil {
		t.Fatalf("open spec: %v", err)
	}
	defer func() { _ = file.Close() }()

	operations := make(map[string]struct{})
	inPaths := false
	currentPath := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " ")
		switch {
		case trimmed == "paths:":
			inPaths = true
		case inPaths && trimmed != "" && !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "#"):
			inPaths = false // следующая top-level секция (components: и т.п.)
		case inPaths && strings.HasPrefix(trimmed, "  /") && strings.HasSuffix(trimmed, ":"):
			currentPath = strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
		case inPaths && currentPath != "" && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     "):
			key := strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
			if method, ok := specMethods[key]; ok {
				operations[fmt.Sprintf("%s %s", method, currentPath)] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(operations) == 0 {
		t.Fatal("no operations parsed from the spec — did its layout change?")
	}
	return operations
}
