package app

import "testing"

func TestParseDefaultOrderTemplates(t *testing.T) {
	templates := parseDefaultOrderTemplates(`
КОКРОКИ
15542 Кокрок с картофелем 0
15544 Кокрок с творогом 0

САМСА И УЧПУЧМАК
15646 Самса с курицей 0
`)
	if len(templates) != 2 {
		t.Fatalf("templates = %d, want 2", len(templates))
	}
	if templates[0].Theme != defaultTemplateTheme {
		t.Fatalf("theme = %q, want %q", templates[0].Theme, defaultTemplateTheme)
	}
	if templates[0].Name != "КОКРОКИ" {
		t.Fatalf("name = %q, want КОКРОКИ", templates[0].Name)
	}
	if templates[0].Body != "КОКРОКИ\n15542 Кокрок с картофелем 0\n15544 Кокрок с творогом 0" {
		t.Fatalf("body = %q", templates[0].Body)
	}
	if templates[1].Name != "САМСА И УЧПУЧМАК" {
		t.Fatalf("name = %q, want САМСА И УЧПУЧМАК", templates[1].Name)
	}
}
