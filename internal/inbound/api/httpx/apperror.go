package httpx

import (
	"log/slog"
	"net/http"

	"bakery/internal/pkg/apperr"
)

// statusByKind maps the transport-agnostic error taxonomy to HTTP status codes.
var statusByKind = map[apperr.Kind]int{
	apperr.KindInvalid:      http.StatusBadRequest,
	apperr.KindNotFound:     http.StatusNotFound,
	apperr.KindConflict:     http.StatusConflict,
	apperr.KindUnauthorized: http.StatusUnauthorized,
	apperr.KindForbidden:    http.StatusForbidden,
	apperr.KindInternal:     http.StatusInternalServerError,
}

// WriteAppError maps a use-case error to an HTTP response. If err carries an
// *apperr.Error, its Kind selects the status and its safe Message is returned;
// otherwise the error is logged and fallback is returned as a 500.
func WriteAppError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	if ae := apperr.As(err); ae != nil {
		status, ok := statusByKind[ae.Kind]
		if !ok {
			status = http.StatusInternalServerError
		}
		if status >= http.StatusInternalServerError {
			slog.ErrorContext(r.Context(), "request failed", "code", ae.Code, "error", err)
		}
		WriteError(w, status, ae.Message)
		return
	}
	slog.ErrorContext(r.Context(), "request failed", "error", err)
	WriteError(w, http.StatusInternalServerError, fallback)
}
