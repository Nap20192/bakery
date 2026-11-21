package syncuc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"bakery/internal/outbound/iiko"
)

type stubClient struct {
	auth    func(context.Context) error
	logout  func(context.Context) error
	catalog func(context.Context) (*iiko.NomenclatureResponse, error)
	charts  func(context.Context) (*iiko.ChartResultDto, error)
}

func (c stubClient) Auth(ctx context.Context) error {
	if c.auth != nil {
		return c.auth(ctx)
	}
	return nil
}

func (c stubClient) Logout(ctx context.Context) error {
	if c.logout != nil {
		return c.logout(ctx)
	}
	return nil
}

func (c stubClient) ListProductsWithCategories(ctx context.Context) (*iiko.NomenclatureResponse, error) {
	if c.catalog != nil {
		return c.catalog(ctx)
	}
	return &iiko.NomenclatureResponse{}, nil
}

func (c stubClient) AssemblyChartsGetAll(ctx context.Context, _, _ string, _, _ bool) (*iiko.ChartResultDto, error) {
	if c.charts != nil {
		return c.charts(ctx)
	}
	return &iiko.ChartResultDto{}, nil
}

type repositoryFunc func(context.Context, *iiko.NomenclatureResponse, *iiko.ChartResultDto, string) error

func (f repositoryFunc) SaveSnapshot(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto, date string) error {
	return f(ctx, catalog, charts, date)
}

func TestServiceCancellation(t *testing.T) {
	for _, run := range []bool{false, true} {
		for _, stage := range []string{"auth", "fetch", "save", "logout timeout"} {
			name := "SyncOnce/" + stage
			if run {
				name = "Run/" + stage
			}
			t.Run(name, func(t *testing.T) {
				var logs bytes.Buffer
				previous := slog.Default()
				slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
				defer slog.SetDefault(previous)
				started := make(chan struct{}, 2)
				block := func(ctx context.Context) error {
					started <- struct{}{}
					<-ctx.Done()
					return ctx.Err()
				}
				type contextKey struct{}
				ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "sync"))
				defer cancel()
				loggedOut := false
				client := stubClient{logout: func(cleanupCtx context.Context) error {
					loggedOut = true
					if cleanupCtx.Err() != nil || cleanupCtx.Value(contextKey{}) != "sync" {
						t.Error("logout must retain values and be independent of sync cancellation")
					}
					deadline, ok := cleanupCtx.Deadline()
					if remaining := time.Until(deadline); !ok || remaining <= 0 || remaining > 500*time.Millisecond {
						t.Error("cancelled sync logout must have an independent deadline within 500 ms")
					}
					if stage == "logout timeout" {
						<-cleanupCtx.Done()
						return cleanupCtx.Err()
					}
					return nil
				}}
				repo := repositoryFunc(func(ctx context.Context, _ *iiko.NomenclatureResponse, _ *iiko.ChartResultDto, _ string) error {
					if stage != "save" {
						t.Error("cancelled fetch must not save a snapshot")
						return nil
					}
					return block(ctx)
				})
				requests := 1
				switch stage {
				case "auth":
					client.auth = block
				case "fetch", "logout timeout":
					requests = 2
					client.catalog = func(ctx context.Context) (*iiko.NomenclatureResponse, error) {
						return nil, block(ctx)
					}
					client.charts = func(ctx context.Context) (*iiko.ChartResultDto, error) {
						return nil, block(ctx)
					}
				}
				service := NewService(client, repo, time.Hour)
				done := make(chan error, 1)
				go func() {
					if run {
						done <- service.Run(ctx)
						return
					}
					done <- service.SyncOnce(ctx)
				}()
				for range requests {
					select {
					case <-started:
					case <-time.After(time.Second):
						t.Fatal("sync did not reach the cancellation stage")
					}
				}
				cancel()
				select {
				case err := <-done:
					if run && err != nil {
						t.Fatalf("Run returned %v", err)
					}
					if !run && !errors.Is(err, context.Canceled) {
						t.Fatalf("SyncOnce must wrap context.Canceled, got %v", err)
					}
				case <-time.After(750 * time.Millisecond):
					t.Fatal("sync did not stop promptly after cancellation")
				}
				if loggedOut != (stage != "auth") {
					t.Fatalf("logout called: %v, stage: %s", loggedOut, stage)
				}
				if strings.Contains(logs.String(), "level=ERROR") {
					t.Fatalf("cancellation logged as failure: %s", logs.String())
				}
			})
		}
	}
}

func TestSyncOnceCancelsSiblingFetch(t *testing.T) {
	failure := errors.New("fetch failed")
	started := make(chan struct{})
	client := stubClient{
		catalog: func(ctx context.Context) (*iiko.NomenclatureResponse, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		},
		charts: func(context.Context) (*iiko.ChartResultDto, error) {
			<-started
			return nil, failure
		},
	}
	service := NewService(client, nil, time.Hour)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := service.SyncOnce(ctx); !errors.Is(err, failure) {
		t.Fatalf("expected original fetch failure, got %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("sibling fetch waited for parent cancellation")
	}
}

func TestLogoutAllowsRemoteRoundTrip(t *testing.T) {
	for _, cancelSync := range []bool{false, true} {
		name := "success"
		if cancelSync {
			name = "cancelled"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				roundTrip := time.Second
				if cancelSync {
					roundTrip = 100 * time.Millisecond
				}
				loggedOut := false
				client := stubClient{
					catalog: func(context.Context) (*iiko.NomenclatureResponse, error) {
						if cancelSync {
							cancel()
							return nil, ctx.Err()
						}
						return &iiko.NomenclatureResponse{}, nil
					},
					logout: func(cleanupCtx context.Context) error {
						select {
						case <-time.After(roundTrip):
							loggedOut = true
							return nil
						case <-cleanupCtx.Done():
							return cleanupCtx.Err()
						}
					},
				}
				repo := repositoryFunc(func(context.Context, *iiko.NomenclatureResponse, *iiko.ChartResultDto, string) error {
					return nil
				})
				err := NewService(client, repo, time.Hour).SyncOnce(ctx)
				if cancelSync && !errors.Is(err, context.Canceled) || !cancelSync && err != nil {
					t.Fatalf("unexpected sync result: %v", err)
				}
				if !loggedOut {
					t.Fatalf("logout timed out before a %s round trip could complete", roundTrip)
				}
			})
		})
	}
}

func TestLogoutTimeout(t *testing.T) {
	for _, stage := range []string{"success", "cancelled before logout", "cancelled during logout"} {
		t.Run(stage, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				type contextKey struct{}
				ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "sync"))
				defer cancel()
				wantTimeout := 5 * time.Second
				if stage != "success" {
					wantTimeout = 500 * time.Millisecond
				}
				client := stubClient{logout: func(cleanupCtx context.Context) error {
					if cleanupCtx.Err() != nil || cleanupCtx.Value(contextKey{}) != "sync" {
						t.Error("logout must retain values and be independent of sync cancellation")
					}
					if stage == "cancelled during logout" {
						time.Sleep(time.Second)
						cancel()
					}
					start := time.Now()
					<-cleanupCtx.Done()
					if elapsed := time.Since(start); elapsed != wantTimeout {
						t.Errorf("logout waited %s, want %s", elapsed, wantTimeout)
					}
					return cleanupCtx.Err()
				}}
				repo := repositoryFunc(func(context.Context, *iiko.NomenclatureResponse, *iiko.ChartResultDto, string) error {
					if stage == "cancelled before logout" {
						cancel()
						return ctx.Err()
					}
					return nil
				})
				err := NewService(client, repo, time.Hour).SyncOnce(ctx)
				if stage == "cancelled before logout" {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("expected context.Canceled, got %v", err)
					}
				} else if err != nil {
					t.Fatalf("unexpected sync result: %v", err)
				}
			})
		})
	}
}
