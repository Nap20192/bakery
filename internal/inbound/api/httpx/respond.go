// Package httpx holds the shared HTTP transport primitives used by every
// service's infra/http delivery adapter: response writers, common DTOs, the
// authenticated viewer context, and the authentication middleware.
package httpx

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ErrorResponse is the standard JSON error body.
type ErrorResponse struct {
	Error string `json:"error"`
}

// DepartmentResponse is the shared department projection embedded by several
// services (orders carry departments, the department and admin lists return
// them directly).
type DepartmentResponse struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// WriteJSON encodes payload as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Warn("write json response failed", "error", err)
	}
}

// WriteError is a shorthand for an ErrorResponse body.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// Trim trims surrounding whitespace.
func Trim(value string) string {
	return strings.TrimSpace(value)
}

// ParseRequestDate parses an RFC3339 or YYYY-MM-DD date; empty input yields the
// zero time without error.
func ParseRequestDate(raw string) (time.Time, error) {
	raw = Trim(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date format: %q", raw)
}

// UniqueQueryValues returns trimmed, de-duplicated, non-empty query values.
func UniqueQueryValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = Trim(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
