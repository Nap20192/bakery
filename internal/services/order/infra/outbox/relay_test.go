package orderoutbox

import (
	"context"
	"errors"
	"testing"

	"bakery/internal/outbound/db/dbtest"
	sqlc "bakery/internal/outbound/db/sqlc"
)

type recordingHandler struct {
	err   error
	types []string
}

func (h *recordingHandler) HandleOrderEvent(_ context.Context, eventType string, _ []byte) error {
	if h.err != nil {
		return h.err
	}
	h.types = append(h.types, eventType)
	return nil
}

func TestRelayDeliversInOrderAndWaitsWhileHandlerNotReady(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	queries := sqlc.New(db)
	for _, eventType := range []string{"order.created", "order.updated"} {
		if _, err := queries.InsertOrderOutboxEvent(ctx, sqlc.InsertOrderOutboxEventParams{AggregateID: "A", EventType: eventType, Payload: "{}"}); err != nil {
			t.Fatal(err)
		}
	}
	pending := func() int64 {
		rows, err := queries.ListUnpublishedOrderOutboxEvents(ctx, 100)
		if err != nil {
			t.Fatal(err)
		}
		return int64(len(rows))
	}

	waiting := &recordingHandler{err: ErrHandlerNotReady}
	if err := NewRelay(queries, waiting, 0).drain(ctx); err != nil {
		t.Fatalf("drain while handler not ready = %v, want nil", err)
	}
	if got := pending(); got != 2 {
		t.Fatalf("pending = %d after not-ready drain, want 2", got)
	}

	failing := &recordingHandler{err: errors.New("boom")}
	if err := NewRelay(queries, failing, 0).drain(ctx); err == nil {
		t.Fatal("drain with failing handler = nil, want error")
	}

	ready := &recordingHandler{}
	if err := NewRelay(queries, ready, 0).drain(ctx); err != nil {
		t.Fatal(err)
	}
	if len(ready.types) != 2 || ready.types[0] != "order.created" || ready.types[1] != "order.updated" {
		t.Fatalf("delivered = %v", ready.types)
	}
	if got := pending(); got != 0 {
		t.Fatalf("pending = %d after delivery, want 0", got)
	}
}
