package synchttp

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type stubSync struct {
	started chan context.Context
	cleanup func()
}

func TestManualSyncDeadlineIsFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer
		previous := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
		defer slog.SetDefault(previous)
		service := stubSync{started: make(chan context.Context, 1)}
		handler := New(service, t.Context())
		response := httptest.NewRecorder()
		handler.handleSync(response, httptest.NewRequest(http.MethodPost, "/admin/iiko/sync", nil))
		ctx := <-service.started
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		if ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("expected manual sync deadline, got %v", ctx.Err())
		}
		if !strings.Contains(logs.String(), "level=ERROR msg=\"manual iiko sync failed\"") {
			t.Fatalf("manual sync deadline must be logged as failure: %s", logs.String())
		}
	})
}

func (s stubSync) Run(context.Context) error { return nil }

func (s stubSync) SyncOnce(ctx context.Context) error {
	s.started <- ctx
	<-ctx.Done()
	if s.cleanup != nil {
		s.cleanup()
	}
	return fmt.Errorf("sync: %w", ctx.Err())
}

func TestManualSyncShutdownWaitsForCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		appCtx, cancel := context.WithCancel(t.Context())
		defer cancel()
		cleanupStarted := make(chan struct{})
		releaseCleanup := make(chan struct{})
		service := stubSync{
			started: make(chan context.Context, 1),
			cleanup: func() {
				close(cleanupStarted)
				<-releaseCleanup
			},
		}
		handler := New(service, appCtx)
		request := httptest.NewRequest(http.MethodPost, "/admin/iiko/sync", nil)
		response := httptest.NewRecorder()
		handler.handleSync(response, request)
		if response.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", response.Code)
		}
		<-service.started
		cancel()
		<-cleanupStarted
		stopped := make(chan struct{})
		go func() {
			handler.Shutdown()
			close(stopped)
		}()
		synctest.Wait()
		select {
		case <-stopped:
			t.Error("shutdown returned before manual sync cleanup completed")
		default:
		}
		response = httptest.NewRecorder()
		handler.handleSync(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 during shutdown, got %d", response.Code)
		}
		close(releaseCleanup)
		<-stopped
	})
}

func TestManualSyncRejectsAfterShutdown(t *testing.T) {
	for _, cancelApp := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		service := stubSync{started: make(chan context.Context, 1)}
		handler := New(service, ctx)
		if cancelApp {
			cancel()
		} else {
			handler.Shutdown()
		}
		response := httptest.NewRecorder()
		handler.handleSync(response, httptest.NewRequest(http.MethodPost, "/admin/iiko/sync", nil))
		cancel()
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 after shutdown, got %d", response.Code)
		}
		select {
		case <-service.started:
			t.Fatal("manual sync started after shutdown")
		default:
		}
	}
}

func TestManualSyncLifecycle(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	appCtx, shutdown := context.WithCancel(t.Context())
	defer shutdown()
	requestCtx, finishRequest := context.WithCancel(t.Context())
	defer finishRequest()
	service := stubSync{started: make(chan context.Context, 1)}
	handler := New(service, appCtx)
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(requestCtx, http.MethodPost, "/admin/iiko/sync", nil)
	handler.handleSync(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}
	var syncCtx context.Context
	select {
	case syncCtx = <-service.started:
	case <-time.After(time.Second):
		t.Fatal("manual sync did not start")
	}
	finishRequest()
	if syncCtx.Err() != nil {
		t.Fatal("finishing the HTTP request cancelled the manual sync")
	}
	deadline, ok := syncCtx.Deadline()
	if !ok || time.Until(deadline) > 10*time.Minute {
		t.Fatal("manual sync must have a ten-minute timeout")
	}
	conflict := httptest.NewRecorder()
	handler.handleSync(conflict, request)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected 409 during sync, got %d", conflict.Code)
	}
	shutdown()
	done := make(chan struct{})
	go func() {
		handler.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("manual sync did not stop promptly on shutdown")
	}
	if syncCtx.Err() != context.Canceled {
		t.Fatalf("expected shutdown cancellation, got %v", syncCtx.Err())
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Fatalf("cancellation logged as failure: %s", logs.String())
	}
}
