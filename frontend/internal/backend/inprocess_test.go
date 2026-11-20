package backend

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"bakery/frontend/internal/application"
)

func TestInProcessClientServesRequestThroughHandler(t *testing.T) {
	t.Parallel()
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/orders" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("category_id") != "7" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("authorization = %q", got)
		}
		if r.Body == nil {
			t.Error("server request body is nil")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[],"page":1,"limit":100,"offset":0,"total":0,"total_pages":0}`)
	})
	client := NewInProcess(api, time.Second)

	page, err := client.Orders(context.Background(), "Bearer token", application.OrderFilters{Page: 1, Limit: 100, CategoryID: 7})
	if err != nil {
		t.Fatalf("Orders: %v", err)
	}
	if page.Page != 1 || page.Limit != 100 {
		t.Fatalf("page = %+v", page)
	}
}

func TestInProcessClientDecodesAPIError(t *testing.T) {
	t.Parallel()
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":"Конфликт"}`)
	})
	client := NewInProcess(api, time.Second)

	err := client.Health(context.Background())
	if got := application.StatusOf(err); got != http.StatusConflict {
		t.Fatalf("status = %d, err = %v", got, err)
	}
	if got := application.MessageOf(err, "fallback"); got != "Конфликт" {
		t.Fatalf("message = %q", got)
	}
}
