package web

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"bakery/internal/inbound/api/contract"
)

// UX invariants checked on every rendered view: things a phone user or a
// screen-reader user notices at once and that silently regress in templates.

func renderedViews(t *testing.T) map[string][]*html.Node {
	t.Helper()
	srv, pages := viewFixtures(t)
	views := make(map[string][]*html.Node)
	for _, value := range pages {
		var buffer bytes.Buffer
		if err := srv.templates.ExecuteTemplate(&buffer, "layout", value); err != nil {
			t.Fatalf("render %s: %v", value.View, err)
		}
		doc, err := html.Parse(&buffer)
		if err != nil {
			t.Fatalf("parse %s: %v", value.View, err)
		}
		views[value.View] = append(views[value.View], doc)
	}
	return views
}

func walk(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, visit)
	}
}

func attr(node *html.Node, name string) (string, bool) {
	for _, a := range node.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

func hasClass(node *html.Node, class string) bool {
	value, _ := attr(node, "class")
	for _, item := range strings.Fields(value) {
		if item == class {
			return true
		}
	}
	return false
}

func text(node *html.Node) string {
	var builder strings.Builder
	walk(node, func(n *html.Node) {
		if n.Type == html.TextNode {
			builder.WriteString(n.Data)
		}
	})
	return strings.TrimSpace(builder.String())
}

func insideLabel(node *html.Node) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && parent.Data == "label" {
			return true
		}
	}
	return false
}

func nonEmptyAttr(node *html.Node, names ...string) bool {
	for _, name := range names {
		if value, ok := attr(node, name); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func TestEveryControlHasAnAccessibleName(t *testing.T) {
	t.Parallel()
	for view, docs := range renderedViews(t) {
		for _, doc := range docs {
			labelled := make(map[string]bool)
			walk(doc, func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "label" {
					if target, ok := attr(n, "for"); ok {
						labelled[target] = true
					}
				}
			})
			walk(doc, func(n *html.Node) {
				if n.Type != html.ElementNode {
					return
				}
				switch n.Data {
				case "input", "select", "textarea":
					kind, _ := attr(n, "type")
					if kind == "hidden" || kind == "submit" || kind == "button" {
						return
					}
					id, _ := attr(n, "id")
					if insideLabel(n) || labelled[id] || nonEmptyAttr(n, "aria-label", "aria-labelledby") {
						return
					}
					name, _ := attr(n, "name")
					t.Errorf("%s: <%s name=%q> has no label or aria-label", view, n.Data, name)
				case "button":
					if text(n) == "" && !nonEmptyAttr(n, "aria-label", "title") {
						t.Errorf("%s: button without text or aria-label", view)
					}
				case "a":
					if _, ok := attr(n, "href"); ok && text(n) == "" && !nonEmptyAttr(n, "aria-label", "title") {
						href, _ := attr(n, "href")
						t.Errorf("%s: link %q without text or aria-label", view, href)
					}
				}
			})
		}
	}
}

func TestIDsAreUniquePerPage(t *testing.T) {
	t.Parallel()
	for view, docs := range renderedViews(t) {
		for _, doc := range docs {
			seen := make(map[string]bool)
			walk(doc, func(n *html.Node) {
				if n.Type != html.ElementNode {
					return
				}
				if id, ok := attr(n, "id"); ok {
					if seen[id] {
						t.Errorf("%s: duplicate id %q", view, id)
					}
					seen[id] = true
				}
			})
		}
	}
}

func TestSteppersNeverSubmitAndNameTheirValue(t *testing.T) {
	t.Parallel()
	found := 0
	for view, docs := range renderedViews(t) {
		for _, doc := range docs {
			walk(doc, func(n *html.Node) {
				if n.Type != html.ElementNode || !hasClass(n, "stepper") {
					return
				}
				found++
				var buttons, inputs int
				walk(n, func(child *html.Node) {
					if child.Type != html.ElementNode {
						return
					}
					switch child.Data {
					case "button":
						buttons++
						if kind, _ := attr(child, "type"); kind != "button" {
							t.Errorf("%s: stepper button type=%q would submit the form", view, kind)
						}
						if !nonEmptyAttr(child, "aria-label") {
							t.Errorf("%s: stepper button without aria-label", view)
						}
					case "input":
						inputs++
						if mode, _ := attr(child, "inputmode"); mode != "numeric" {
							t.Errorf("%s: stepper input inputmode=%q, want numeric keyboard", view, mode)
						}
						if minimum, _ := attr(child, "min"); minimum != "0" {
							t.Errorf("%s: stepper input min=%q, want 0", view, minimum)
						}
					}
				})
				if buttons != 2 || inputs != 1 {
					t.Errorf("%s: stepper has %d buttons and %d inputs, want 2 and 1", view, buttons, inputs)
				}
			})
		}
	}
	if found == 0 {
		t.Fatal("no steppers rendered: the order form, production editor and calculator must use them")
	}
}

func TestMobileTabBarHasAtMostFiveTabs(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"shop", "baker", "admin"} {
		doc, err := html.Parse(strings.NewReader(renderLayout(t, page{View: "me", Viewer: &contract.Me{Role: role, TelegramUsername: "u"}})))
		if err != nil {
			t.Fatal(err)
		}
		var tabs, creates, secondary int
		var profileLabel string
		walk(doc, func(n *html.Node) {
			if n.Type != html.ElementNode || n.Data != "a" || n.Parent == nil {
				return
			}
			if id, _ := attr(n.Parent, "id"); id != "main-nav" {
				return
			}
			if hasClass(n, "nav-secondary") {
				secondary++
				return
			}
			tabs++
			if hasClass(n, "nav-create") {
				creates++
			}
			if href, _ := attr(n, "href"); href == "/me" {
				profileLabel = text(n)
			}
		})
		if tabs > 5 {
			t.Errorf("%s: %d tabs in the phone bar, want at most 5", role, tabs)
		}
		if creates != 1 {
			t.Errorf("%s: %d «Создать» tabs, want 1", role, creates)
		}
		if role == "admin" && (secondary != 3 || !strings.Contains(profileLabel, "Ещё")) {
			t.Errorf("admin: %d secondary sections, profile label %q; want 3 moved to «Ещё»", secondary, profileLabel)
		}
		if role != "admin" && secondary != 0 {
			t.Errorf("%s: %d admin-only sections in the bar", role, secondary)
		}
	}
}

func TestPortraitMatrixCellsNameTheirShop(t *testing.T) {
	t.Parallel()
	cells := 0
	for _, doc := range renderedViews(t)["orders"] {
		walk(doc, func(n *html.Node) {
			if n.Type != html.ElementNode || n.Data != "td" {
				return
			}
			cells++
			named := false
			walk(n, func(child *html.Node) {
				if child.Type == html.ElementNode && hasClass(child, "matrix-cell-shop") && text(child) != "" {
					named = true
				}
			})
			if !named {
				t.Error("matrix cell without its shop name: the portrait day feed would lose the shop axis")
			}
		})
	}
	if cells == 0 {
		t.Fatal("no matrix cells rendered")
	}
}

func TestStylesheetKeepsTouchTargets(t *testing.T) {
	t.Parallel()
	css, err := frontendFiles.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	src := string(css)
	for selector, want := range map[string][]string{
		`.stepper-btn {`:             {"width: 44px", "height: 44px"},
		`.stepper .quantity-input {`: {"min-height: 44px"},
		`.main-nav a { flex: 1 1 0;`: {"min-height: 52px"},
	} {
		index := strings.Index(src, selector)
		if index < 0 {
			t.Errorf("app.css lost rule %q", selector)
			continue
		}
		end := strings.Index(src[index:], "}")
		block := src[index : index+end]
		for _, declaration := range want {
			if !strings.Contains(block, declaration) {
				t.Errorf("%s lost %q", selector, declaration)
			}
		}
	}
	if !regexp.MustCompile(`@media \(max-width: 640px\)[\s\S]*\.button \{ min-height: 44px; \}`).MatchString(src) {
		t.Error("phone buttons are no longer at least 44px tall")
	}
	if !regexp.MustCompile(`@media \(max-width: 900px\)[\s\S]*\.order-preview-action \{ width: 44px; height: 44px;`).MatchString(src) {
		t.Error("the card's «Обзор» button is below a 44px touch target on phones")
	}
}
