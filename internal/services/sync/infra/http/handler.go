// Package synchttp is the HTTP delivery adapter of the iiko sync service: an
// admin-only button to trigger a sync outside the background ticker.
package synchttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"bakery/internal/inbound/api/httpx"
	syncuc "bakery/internal/services/sync/usecase/sync"
)

// Handler triggers a one-off iiko sync.
type Handler struct {
	sync     syncuc.UseCase
	ctx      context.Context
	mu       sync.Mutex
	done     chan struct{}
	stopping bool
}

func New(sync syncuc.UseCase, ctx context.Context) *Handler {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Handler{sync: sync, ctx: ctx}
}

// RegisterRoutes wires the sync trigger behind the admin role.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("POST /admin/iiko/sync", auth.RequireAdmin(http.HandlerFunc(h.handleSync)))
}

func (h *Handler) Shutdown() {
	h.mu.Lock()
	h.stopping = true
	done := h.done
	h.mu.Unlock()
	if done != nil {
		<-done
	}
}

// handleSync starts the sync in the background and returns immediately — a full
// iiko snapshot fetch is too slow to hold an HTTP request open.
func (h *Handler) handleSync(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.stopping || h.ctx.Err() != nil {
		h.mu.Unlock()
		httpx.WriteError(w, http.StatusServiceUnavailable, "Приложение останавливается.")
		return
	}
	if h.done != nil {
		h.mu.Unlock()
		httpx.WriteError(w, http.StatusConflict, "Синхронизация уже идёт.")
		return
	}
	h.done = make(chan struct{})
	h.mu.Unlock()
	go func() {
		defer func() {
			h.mu.Lock()
			close(h.done)
			h.done = nil
			h.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(h.ctx, 10*time.Minute)
		defer cancel()
		if err := h.sync.SyncOnce(ctx); err != nil {
			if h.ctx.Err() != nil && errors.Is(err, context.Canceled) {
				slog.InfoContext(ctx, "manual iiko sync stopped", "error", err)
				return
			}
			slog.ErrorContext(ctx, "manual iiko sync failed", "error", err)
			return
		}
		slog.InfoContext(ctx, "manual iiko sync finished")
	}()
	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
