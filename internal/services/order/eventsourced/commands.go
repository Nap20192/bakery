package eventsourced

import (
	"context"
	"errors"
	"fmt"

	"bakery/internal/pkg/apperr"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/core"
)

var ErrOrderNotFound = apperr.NotFound("order.not_found", "Заказ не найден.")

// Commands — командная сторона CQRS: загружает агрегат из потока событий,
// выполняет команду и сохраняет новые события, затем синхронно проецирует их
// в read model. Writer обязан быть идемпотентным: при падении проекции
// события уже в store, и их догоняет фоновый прогон той же проекции.
type Commands struct {
	store  core.EventStore
	writer ReadModelWriter
}

func NewCommands(store core.EventStore, writer ReadModelWriter) *Commands {
	return &Commands{store: store, writer: writer}
}

// CreateOrder создаёт агрегат. Номер, категория и нормализованные даты
// приходят готовыми: генерация номера (счётчик) и проверки каталога —
// ответственность вызывающего usecase, инварианты состава — агрегата.
func (c *Commands) CreateOrder(ctx context.Context, data Created) (*Order, error) {
	order, err := NewOrder(data)
	if err != nil {
		return nil, err
	}
	if err := c.commit(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (c *Commands) UpdateOrder(ctx context.Context, number string, data ItemsUpdated) (*Order, error) {
	order, err := c.load(ctx, number)
	if err != nil {
		return nil, err
	}
	if err := order.UpdateItems(data); err != nil {
		return nil, err
	}
	if err := c.commit(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (c *Commands) CancelOrder(ctx context.Context, number, byUsername string) (*Order, error) {
	order, err := c.load(ctx, number)
	if err != nil {
		return nil, err
	}
	order.Cancel(byUsername)
	if err := c.commit(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (c *Commands) RestoreOrder(ctx context.Context, number, byUsername string) (*Order, error) {
	order, err := c.load(ctx, number)
	if err != nil {
		return nil, err
	}
	order.Restore(byUsername)
	if err := c.commit(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (c *Commands) RecordProduction(ctx context.Context, number string, sheetID int64, byUsername string, items []ProducedItem) (*Order, error) {
	order, err := c.load(ctx, number)
	if err != nil {
		return nil, err
	}
	if err := order.RecordProduction(sheetID, byUsername, items); err != nil {
		return nil, err
	}
	if err := c.commit(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (c *Commands) ClearProduction(ctx context.Context, number, byUsername string) (*Order, error) {
	order, err := c.load(ctx, number)
	if err != nil {
		return nil, err
	}
	order.ClearProduction(byUsername)
	if err := c.commit(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (c *Commands) load(ctx context.Context, number string) (*Order, error) {
	order := &Order{}
	if err := aggregate.Load(ctx, c.store, number, order); err != nil {
		if errors.Is(err, eventsourcing.ErrAggregateNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("load order %s: %w", number, err)
	}
	return order, nil
}

// commit сохраняет несохранённые события агрегата и синхронно применяет их к
// read model. Идемпотентные команды без изменений (len(pending)==0) — no-op.
func (c *Commands) commit(ctx context.Context, order *Order) error {
	pending := order.Events()
	if len(pending) == 0 {
		return nil
	}
	if err := aggregate.Save(c.store, order); err != nil {
		return fmt.Errorf("save order events: %w", err)
	}
	for _, event := range pending {
		if err := c.writer.Apply(ctx, event); err != nil {
			// События уже в store (источник истины). Read model догонит их
			// фоновым прогоном проекции; наружу отдаём ошибку, чтобы вызов
			// не выглядел успешным при отставшем чтении.
			return fmt.Errorf("project order event %s: %w", event.Reason(), err)
		}
	}
	return nil
}
