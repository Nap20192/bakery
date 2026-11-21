package iiko

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func cancellationCalls() []struct {
	name string
	call func(context.Context, *Client) error
} {
	return []struct {
		name string
		call func(context.Context, *Client) error
	}{
		{"auth", func(ctx context.Context, c *Client) error { return c.Auth(ctx) }},
		{"logout", func(ctx context.Context, c *Client) error { return c.Logout(ctx) }},
		{"products", func(ctx context.Context, c *Client) error {
			_, err := c.ListProducts(ctx)
			return err
		}},
		{"nomenclature", func(ctx context.Context, c *Client) error {
			_, err := c.ListProductsWithCategories(ctx)
			return err
		}},
		{"all charts", func(ctx context.Context, c *Client) error {
			_, err := c.AssemblyChartsGetAll(ctx, "2026-11-21", "2026-12-21", false, true)
			return err
		}},
		{"chart by id", func(ctx context.Context, c *Client) error {
			_, err := c.AssemblyChartByID(ctx, "chart")
			return err
		}},
		{"assembled chart", func(ctx context.Context, c *Client) error {
			_, err := c.AssemblyChartsGetAssembled(ctx, "2026-11-21", "product", "department")
			return err
		}},
		{"prepared chart", func(ctx context.Context, c *Client) error {
			_, err := c.AssemblyChartsGetPrepared(ctx, "2026-11-21", "product", "department")
			return err
		}},
	}
}

func TestClientCancellation(t *testing.T) {
	for _, tt := range cancellationCalls() {
		t.Run(tt.name, func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			defer server.CloseClientConnections()
			host, port, err := net.SplitHostPort(server.Listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewClient("login", "password", NewApi(host, port))
			if err != nil {
				t.Fatal(err)
			}
			client.client = server.Client()
			client.key = "session"
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tt.call(ctx, client) }()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("request did not reach the server")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected context.Canceled, got %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("request did not stop within one second of cancellation")
			}
		})
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientRequestContext(t *testing.T) {
	for _, tt := range cancellationCalls() {
		t.Run(tt.name, func(t *testing.T) {
			type contextKey struct{}
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "request"))
			defer cancel()
			client, err := NewClient("login", "password", NewApi("iiko.invalid", "443"))
			if err != nil {
				t.Fatal(err)
			}
			client.key = "session"
			client.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Context().Value(contextKey{}) != "request" {
					t.Error("request did not carry the caller's context")
				}
				cancel()
				return nil, r.Context().Err()
			})
			if err := tt.call(ctx, client); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected wrapped context.Canceled, got %v", err)
			}
		})
	}
}
