package eventsourced

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
)

var registerOnce sync.Once

func newTestStore() *memory.Memory {
	registerOnce.Do(RegisterAggregates)
	return memory.Create()
}

func testItems() []Item {
	return []Item{
		{Code: "15693", ProductName: "Багет Особый", Quantity: 10},
		{Code: "20955", ProductName: "Чиабатта", Quantity: 5, ReservedQuantity: 2},
	}
}

func TestOrderLifecycleRebuildsFromEvents(t *testing.T) {
	store := newTestStore()

	order, err := NewOrder(Created{Number: "Г.Х.09.07.26.001", CreatedByUsername: "shop", Items: testItems()})
	if err != nil {
		t.Fatalf("NewOrder: %v", err)
	}
	if err := order.UpdateItems(ItemsUpdated{Items: testItems()[:1], ChangedByUsername: "shop"}); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}
	order.Cancel("baker")
	order.Restore("baker")
	if err := aggregate.Save(store, order); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Состояние восстанавливается только из потока событий.
	loaded := &Order{}
	if err := aggregate.Load(context.Background(), store, "Г.Х.09.07.26.001", loaded); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Number != "Г.Х.09.07.26.001" || loaded.Cancelled || len(loaded.Items) != 1 {
		t.Fatalf("loaded = %+v", loaded)
	}
	if loaded.Version() != 4 {
		t.Fatalf("version = %d, want 4 (created+updated+cancelled+restored)", loaded.Version())
	}
}

func TestOrderInvariants(t *testing.T) {
	if _, err := NewOrder(Created{Number: "A"}); !errors.Is(err, ErrNoItems) {
		t.Fatalf("empty items: %v", err)
	}
	if _, err := NewOrder(Created{Number: "A", Items: []Item{{ProductName: "Хлеб", Quantity: 1}, {ProductName: "хлеб", Quantity: 2}}}); !errors.Is(err, ErrDuplicateItem) {
		t.Fatalf("duplicate items: %v", err)
	}
	if _, err := NewOrder(Created{Number: "A", Items: []Item{{ProductName: "Хлеб", Quantity: 0}}}); !errors.Is(err, ErrBadQuantity) {
		t.Fatalf("zero quantity: %v", err)
	}

	order, err := NewOrder(Created{Number: "A", Items: testItems()})
	if err != nil {
		t.Fatal(err)
	}
	order.Cancel("x")
	if err := order.UpdateItems(ItemsUpdated{Items: testItems()}); !errors.Is(err, ErrCancelled) {
		t.Fatalf("update cancelled: %v", err)
	}
	// Повторная отмена — no-op без нового события.
	events := len(order.Events())
	order.Cancel("x")
	if len(order.Events()) != events {
		t.Fatal("repeated cancel must not add events")
	}
}

func TestOrderProductionDeviationsOnly(t *testing.T) {
	order, err := NewOrder(Created{Number: "A", Items: testItems()})
	if err != nil {
		t.Fatal(err)
	}

	// Факт, равный заявке по всем позициям, — не отработка.
	err = order.RecordProduction(1, "baker", []ProducedItem{
		{ProductName: "Багет Особый", Quantity: 10},
		{ProductName: "Чиабатта", Quantity: 7}, // 5+2 = заявка
	})
	if !errors.Is(err, ErrProductionNoChanges) {
		t.Fatalf("no changes: %v", err)
	}

	// Отклонение сохраняется; совпавшая позиция отбрасывается.
	if err := order.RecordProduction(1, "baker", []ProducedItem{
		{ProductName: "багет особый", Quantity: 8, Reason: "Подгорело"},
		{ProductName: "Чиабатта", Quantity: 7},
	}); err != nil {
		t.Fatalf("RecordProduction: %v", err)
	}
	if len(order.Produced) != 1 || order.Produced["багет особый"].Reason != "Подгорело" {
		t.Fatalf("produced = %#v", order.Produced)
	}
	if order.Produced["багет особый"].Quantity != 8 {
		t.Fatalf("produced багет = %v, want 8", order.Produced["багет особый"].Quantity)
	}

	// Неизвестная позиция — отказ.
	if err := order.RecordProduction(1, "baker", []ProducedItem{{ProductName: "Неизвестное", Quantity: 1}}); !errors.Is(err, ErrProductionUnknown) {
		t.Fatalf("unknown item: %v", err)
	}

	order.ClearProduction("baker")
	if order.Produced != nil || order.ProductionSheetID != nil {
		t.Fatalf("production must be cleared: %#v", order.Produced)
	}
	// Повторное снятие — no-op.
	events := len(order.Events())
	order.ClearProduction("baker")
	if len(order.Events()) != events {
		t.Fatal("repeated clear must not add events")
	}
}

func TestReadModelProjectionDispatch(t *testing.T) {
	store := newTestStore()
	order, err := NewOrder(Created{Number: "B", Items: testItems()})
	if err != nil {
		t.Fatal(err)
	}
	order.Cancel("x")
	if err := aggregate.Save(store, order); err != nil {
		t.Fatal(err)
	}

	writer := &recordingWriter{}
	projection := NewReadModelProjection(store.All(0, 10), writer)
	projection.RunOnce()

	if len(writer.calls) != 2 || writer.calls[0] != "created:B" || writer.calls[1] != "cancelled:B" {
		t.Fatalf("calls = %#v", writer.calls)
	}
}

type recordingWriter struct {
	calls []string
}

func (w *recordingWriter) ApplyCreated(number string, _ Created) error {
	w.calls = append(w.calls, "created:"+number)
	return nil
}
func (w *recordingWriter) ApplyItemsUpdated(number string, _ ItemsUpdated) error {
	w.calls = append(w.calls, "updated:"+number)
	return nil
}
func (w *recordingWriter) ApplyCancelled(number string, _ Cancelled) error {
	w.calls = append(w.calls, "cancelled:"+number)
	return nil
}
func (w *recordingWriter) ApplyRestored(number string, _ Restored) error {
	w.calls = append(w.calls, "restored:"+number)
	return nil
}
func (w *recordingWriter) ApplyProductionRecorded(number string, _ ProductionRecorded) error {
	w.calls = append(w.calls, "produced:"+number)
	return nil
}
func (w *recordingWriter) ApplyProductionCleared(number string, _ ProductionCleared) error {
	w.calls = append(w.calls, "production_cleared:"+number)
	return nil
}
