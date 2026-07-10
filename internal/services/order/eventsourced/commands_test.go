package eventsourced

import (
	"context"
	"errors"
	"testing"
)

func TestCommandsLifecycleProjectsSynchronously(t *testing.T) {
	store := newTestStore()
	writer := &recordingWriter{}
	commands := NewCommands(store, writer)
	ctx := context.Background()

	if _, err := commands.CreateOrder(ctx, Created{Number: "C1", Items: testItems()}); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if _, err := commands.UpdateOrder(ctx, "C1", ItemsUpdated{Items: testItems()[:1], ChangedByUsername: "shop"}); err != nil {
		t.Fatalf("UpdateOrder: %v", err)
	}
	if _, err := commands.CancelOrder(ctx, "C1", "baker"); err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	// Повторная отмена — no-op: ни события, ни проекции.
	if _, err := commands.CancelOrder(ctx, "C1", "baker"); err != nil {
		t.Fatalf("repeated CancelOrder: %v", err)
	}
	if _, err := commands.RestoreOrder(ctx, "C1", "baker"); err != nil {
		t.Fatalf("RestoreOrder: %v", err)
	}
	if _, err := commands.RecordProduction(ctx, "C1", 7, "baker", []ProducedItem{{ProductName: "Багет Особый", Quantity: 8, Reason: "Упало"}}); err != nil {
		t.Fatalf("RecordProduction: %v", err)
	}
	if _, err := commands.ClearProduction(ctx, "C1", "baker"); err != nil {
		t.Fatalf("ClearProduction: %v", err)
	}

	want := []string{"created:C1", "updated:C1", "cancelled:C1", "restored:C1", "produced:C1", "production_cleared:C1"}
	if len(writer.calls) != len(want) {
		t.Fatalf("calls = %#v, want %#v", writer.calls, want)
	}
	for i, call := range want {
		if writer.calls[i] != call {
			t.Fatalf("call[%d] = %s, want %s", i, writer.calls[i], call)
		}
	}

	// Состояние после команд восстанавливается из store.
	order, err := commands.load(ctx, "C1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if order.Cancelled || order.Produced != nil || len(order.Items) != 1 {
		t.Fatalf("order = %+v", order)
	}
}

func TestCommandsNotFoundAndInvariants(t *testing.T) {
	store := newTestStore()
	commands := NewCommands(store, &recordingWriter{})
	ctx := context.Background()

	if _, err := commands.CancelOrder(ctx, "missing", "x"); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("cancel missing: %v", err)
	}

	if _, err := commands.CreateOrder(ctx, Created{Number: "C2", Items: testItems()}); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.CancelOrder(ctx, "C2", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.UpdateOrder(ctx, "C2", ItemsUpdated{Items: testItems()}); !errors.Is(err, ErrCancelled) {
		t.Fatalf("update cancelled: %v", err)
	}
}

func TestCommandsProjectionFailureSurfacesError(t *testing.T) {
	store := newTestStore()
	failing := &failingWriter{}
	commands := NewCommands(store, failing)

	_, err := commands.CreateOrder(context.Background(), Created{Number: "C3", Items: testItems()})
	if err == nil {
		t.Fatal("projection failure must surface as error")
	}
	// События при этом уже в store — фоновая проекция их догонит.
	if _, err := commands.load(context.Background(), "C3"); err != nil {
		t.Fatalf("events must be saved despite projection failure: %v", err)
	}
}

type failingWriter struct{ recordingWriter }

func (w *failingWriter) ApplyCreated(string, Created) error {
	return errors.New("read model unavailable")
}
