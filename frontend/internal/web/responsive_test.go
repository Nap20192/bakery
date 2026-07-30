package web

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"bakery/frontend/internal/backend"
	"bakery/internal/inbound/api/contract"
)

// These guard the mobile-readiness invariants of the rendered shell: a phone
// that loads the page must get a device-width viewport, the CSS must ship the
// small-screen breakpoints, and the bottom navigation must render. They fail
// loudly if a refactor drops the responsive scaffolding.

func renderLayout(t *testing.T, value page) string {
	t.Helper()
	client := backend.New("http://127.0.0.1:1", time.Second)
	srv, err := newServer(client, client, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	value.CSRF = "csrf"
	value.CurrentPath = "/" + value.View
	value.Today = "2026-07-24"
	var buf bytes.Buffer
	if err := srv.templates.ExecuteTemplate(&buf, "layout", value); err != nil {
		t.Fatalf("render %s: %v", value.View, err)
	}
	return buf.String()
}

func TestLayoutShipsMobileViewport(t *testing.T) {
	t.Parallel()
	html := renderLayout(t, page{View: "me", Viewer: &contract.Me{Role: "shop", TelegramUsername: "u"}})
	if !strings.Contains(html, `name="viewport"`) || !strings.Contains(html, "width=device-width") {
		t.Fatal("layout missing device-width viewport meta — page will not scale on phones")
	}
	// viewport-fit=cover keeps content clear of the iPhone notch / home bar.
	if !strings.Contains(html, "viewport-fit=cover") {
		t.Error("viewport-fit=cover dropped — safe-area insets stop working")
	}
}

func TestAuthedShellRendersBottomNav(t *testing.T) {
	t.Parallel()
	// The login screen has no chrome; every authed page must carry the nav.
	login := renderLayout(t, page{View: "login", Data: loginData{Next: "/orders"}})
	if strings.Contains(login, `class="main-nav"`) {
		t.Error("login page should not render the app nav")
	}
	authed := renderLayout(t, page{View: "me", Viewer: &contract.Me{Role: "shop", TelegramUsername: "u"}})
	if !strings.Contains(authed, `class="app-shell"`) || !strings.Contains(authed, `class="main-nav"`) {
		t.Fatal("authed shell missing app-shell / main-nav")
	}
}

func TestStylesheetShipsMobileBreakpoints(t *testing.T) {
	t.Parallel()
	css, err := frontendFiles.ReadFile("static/app.css")
	if err != nil {
		t.Fatalf("read app.css: %v", err)
	}
	src := string(css)
	// Phone breakpoint plus the overflow guard that stops horizontal scroll.
	for _, want := range []string{"@media (max-width: 640px)", "overflow-x: clip"} {
		if !strings.Contains(src, want) {
			t.Errorf("app.css missing %q — mobile layout regresses", want)
		}
	}
}
