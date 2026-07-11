package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bakery/internal/inbound/api/httpx"
	"bakery/internal/pkg/correlation"
)

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return s.recoverer(s.requestLogger(s.cors(next)))
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowedOrigins := parseAllowedOrigins(s.config.AllowedOrigins)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("http panic recovered", "method", r.Method, "path", r.URL.Path, "panic", recovered)
				httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Correlation id ties the request log to the outbox event and the bot's
		// consumption of it.
		ctx, correlationID := correlation.EnsureID(r.Context())
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r.WithContext(ctx))

		level := slog.LevelInfo
		switch {
		case recorder.status >= http.StatusInternalServerError:
			level = slog.LevelError
		case recorder.status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}
		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int("bytes", recorder.bytes),
			slog.String("duration", time.Since(start).Round(time.Millisecond).String()),
			slog.String("correlation_id", correlationID),
		}
		if message := recorder.errorMessage(); message != "" {
			attrs = append(attrs, slog.String("error", message))
		}
		slog.LogAttrs(r.Context(), level, "http request", attrs...)
	})
}

// errorBodyLimit caps how much of an error response body is kept for logging.
const errorBodyLimit = 1024

// statusRecorder captures the response status, body size, and (for error
// responses) the body itself, so the request log can include the error text.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	bytes   int
	errBody []byte
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	if r.status >= http.StatusBadRequest && len(r.errBody) < errorBodyLimit {
		r.errBody = append(r.errBody, b[:min(n, errorBodyLimit-len(r.errBody))]...)
	}
	return n, err
}

// errorMessage extracts the message from a captured httpx.ErrorResponse body.
// Falls back to the raw body for non-JSON error responses.
func (r *statusRecorder) errorMessage() string {
	if len(r.errBody) == 0 {
		return ""
	}
	var payload httpx.ErrorResponse
	if err := json.Unmarshal(r.errBody, &payload); err == nil && payload.Error != "" {
		return payload.Error
	}
	return strings.TrimSpace(string(r.errBody))
}

func parseAllowedOrigins(raw string) map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}
	return allowed
}

func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[origin]
	return ok
}
