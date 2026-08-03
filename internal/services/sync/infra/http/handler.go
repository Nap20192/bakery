// Package synchttp is the HTTP delivery adapter of the iiko sync service: an
// admin-only button to trigger a sync outside the background ticker.
package synchttp

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"bakery/internal/inbound/api/httpx"
	syncuc "bakery/internal/services/sync/usecase/sync"
)

// Handler triggers a one-off iiko sync.
type Handler struct {
	sync syncuc.UseCase
	// running guards the manual trigger so double-clicks don't stack syncs.
	// ponytail: guards the manual button only; the background ticker can still
	// overlap a manual run — harmless (SaveSnapshot is transactional/idempotent).
	// Share one guard with the ticker if overlap ever matters.
	running atomic.Bool
}

func New(sync syncuc.UseCase) *Handler {
	return &Handler{sync: sync}
}

// RegisterRoutes wires the sync trigger behind the admin role.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("POST /admin/iiko/sync", auth.RequireAdmin(http.HandlerFunc(h.handleSync)))
}

// handleSync starts the sync in the background and returns immediately — a full
// iiko snapshot fetch is too slow to hold an HTTP request open.
func (h *Handler) handleSync(w http.ResponseWriter, r *http.Request) {
	if !h.running.CompareAndSwap(false, true) {
		httpx.WriteError(w, http.StatusConflict, "Синхронизация уже идёт.")
		return
	}
	go func() {
		defer h.running.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := h.sync.SyncOnce(ctx); err != nil {
			slog.ErrorContext(ctx, "manual iiko sync failed", "error", err)
			return
		}
		slog.InfoContext(ctx, "manual iiko sync finished")
	}()
	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
