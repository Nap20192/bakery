package backend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"bakery/frontend/internal/application"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestOrdersBuildsAuthorizedFilteredRequest(t *testing.T) {
	t.Parallel()
	client := New("https://api.example.test/", time.Second)
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s", request.Method)
		}
		if request.URL.Path != "/orders" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("category_id") != "7" || request.URL.Query().Get("limit") != "100" {
			t.Errorf("query = %s", request.URL.RawQuery)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("authorization = %q", got)
		}
		return jsonResponse(http.StatusOK, `{"items":[],"page":1,"limit":100,"offset":0,"total":0,"total_pages":0}`), nil
	})

	page, err := client.Orders(context.Background(), "Bearer token", application.OrderFilters{Page: 1, Limit: 100, CategoryID: 7})
	if err != nil {
		t.Fatalf("Orders: %v", err)
	}
	if page.Page != 1 || page.Limit != 100 {
		t.Fatalf("page = %+v", page)
	}
}

func TestMessageAndStatusSurviveWrapping(t *testing.T) {
	t.Parallel()
	err := errors.New("outer: " + (&application.Error{Status: http.StatusConflict, Message: "Конфликт"}).Error())
	if application.StatusOf(err) != 0 {
		t.Fatal("plain text must not be treated as an API error")
	}

	wrapped := errors.Join(errors.New("context"), &application.Error{Status: http.StatusConflict, Message: "Конфликт"})
	if got := application.StatusOf(wrapped); got != http.StatusConflict {
		t.Fatalf("status = %d", got)
	}
	if got := application.MessageOf(wrapped, "fallback"); got != "Конфликт" {
		t.Fatalf("message = %q", got)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
