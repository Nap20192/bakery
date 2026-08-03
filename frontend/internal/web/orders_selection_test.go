package web

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

// stubQueries answers only the calls the selection page makes. The embedded
// interface keeps it compiling as the port grows; any other call panics loudly.
type stubQueries struct {
	application.Queries
	orders map[string]contract.Order
	errs   map[string]error
}

func (s stubQueries) Me(context.Context, application.Credentials) (contract.Me, error) {
	return contract.Me{Role: "baker"}, nil
}

func (s stubQueries) Order(_ context.Context, _ application.Credentials, number string) (contract.Order, error) {
	if err := s.errs[number]; err != nil {
		return contract.Order{}, err
	}
	return s.orders[number], nil
}

func TestSelectionPageDropsOrdersThatCannotJoinABatch(t *testing.T) {
	t.Parallel()
	sheetID := int64(24)
	category := contract.Category{ID: 1, Name: "Хлеб"}
	orders := map[string]contract.Order{
		"free":      {Number: "free", Category: &category},
		"produced":  {Number: "produced", Category: &category, ProductionSheetID: &sheetID},
		"cancelled": {Number: "cancelled", Category: &category, Cancelled: true},
	}

	srv, err := newServer(stubQueries{
		orders: orders,
		errs: map[string]error{
			"missing": &application.Error{Status: http.StatusNotFound, Message: "заказ не найден"},
		},
	}, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	request := httptest.NewRequest("GET", "/orders/selection?order=free&order=produced&order=cancelled&order=missing", nil)
	request.AddCookie(&http.Cookie{
		Name:     sessionCookie,
		Value:    base64.RawURLEncoding.EncodeToString([]byte("Bearer token")),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	response := httptest.NewRecorder()
	srv.orderSelectionPage(response, request)

	if response.Code != 200 {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, number := range []string{"produced", "cancelled", "missing"} {
		if strings.Contains(body, `<strong>`+number+`</strong>`) {
			t.Errorf("%s reached the batch; it already has a sheet or is cancelled", number)
		}
	}
	if !strings.Contains(body, `<strong>free</strong>`) {
		t.Error("the selectable order is missing from the batch")
	}
	if !strings.Contains(body, "Не вошли в партию") {
		t.Error("dropped orders were not reported to the user")
	}
	if !strings.Contains(body, `data-selection-category="1"`) {
		t.Error("selection page does not expose the category needed to resume the batch")
	}
}

func TestSelectionScriptRestoresSavedOrdersBeforeClearingThem(t *testing.T) {
	t.Parallel()
	source, err := frontendFiles.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	script := string(source)
	restore := strings.Index(script, "window.location.replace(selectionURL(saved))")
	clear := strings.Index(script, "storeSelection(current)")
	if restore < 0 || clear < 0 || restore > clear {
		t.Fatal("empty selection page clears saved orders before restoring its missing query")
	}
}

func TestSelectionLinksBypassHTMXBoost(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"templates/pages/orders.html", "templates/pages/orders_table.html"} {
		source, err := frontendFiles.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(source), `hx-boost="false"`) {
			t.Errorf("%s lets HTMX drop repeated order parameters", path)
		}
	}
}
