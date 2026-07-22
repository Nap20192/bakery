package web

import (
	"context"
	"encoding/base64"
	"io"
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
}

func (s stubQueries) Me(context.Context, application.Credentials) (contract.Me, error) {
	return contract.Me{Role: "baker"}, nil
}

func (s stubQueries) Order(_ context.Context, _ application.Credentials, number string) (contract.Order, error) {
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

	srv, err := newServer(stubQueries{orders: orders}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	request := httptest.NewRequest("GET", "/orders/selection?order=free&order=produced&order=cancelled", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: base64.RawURLEncoding.EncodeToString([]byte("Bearer token"))})
	response := httptest.NewRecorder()
	srv.orderSelectionPage(response, request)

	if response.Code != 200 {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, number := range []string{"produced", "cancelled"} {
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
}
