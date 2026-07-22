// Package backend is the outbound HTTP adapter for the bakery JSON API.
//
// The web UI runs as its own service and never touches the database: every
// read and write goes through the same public API the Telegram Mini App used
// to call from the browser. Credentials are passed per request instead of
// being stored on the client, because one process serves many visitors.
package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

// Client talks to the bakery API over HTTP.
type Client struct {
	baseURL string
	http    *http.Client
}

var (
	_ application.QueryGateway   = (*Client)(nil)
	_ application.CommandGateway = (*Client)(nil)
)

// New returns a client for the API rooted at baseURL.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func validCredentials(cred application.Credentials) bool {
	return strings.TrimSpace(string(cred)) != ""
}

type request struct {
	method string
	path   string
	query  url.Values
	body   any
	// out receives the decoded JSON response; nil discards the body.
	out any
}

func (c *Client) do(ctx context.Context, cred application.Credentials, req request) error {
	endpoint := c.baseURL + req.path
	if len(req.query) > 0 {
		endpoint += "?" + req.query.Encode()
	}

	var payload io.Reader
	if req.body != nil {
		encoded, err := json.Marshal(req.body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, endpoint, payload)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	if req.body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if validCredentials(cred) {
		httpReq.Header.Set("Authorization", string(cred))
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call %s %s: %w", req.method, req.path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return decodeError(resp)
	}
	if req.out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(req.out); err != nil {
		return fmt.Errorf("decode %s %s response: %w", req.method, req.path, err)
	}
	return nil
}

func decodeError(resp *http.Response) error {
	var body contract.Error
	// A non-JSON error body (proxy error page, panic) leaves Error empty; the
	// caller substitutes its own fallback text.
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&body)
	return &application.Error{Status: resp.StatusCode, Message: strings.TrimSpace(body.Error)}
}

// Health reports whether the API is reachable. Used by the frontend's own
// health endpoint so a deploy fails loudly when the backend is misconfigured.
func (c *Client) Health(ctx context.Context) error {
	return c.do(ctx, "", request{method: http.MethodGet, path: "/health"})
}
