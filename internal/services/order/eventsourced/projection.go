package eventsourced

import (
	"context"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
)

// ReadModelWriter — порт проекции: применяет событие заказа к read model
// (существующие таблицы orders/order_items). Реализация обязана быть
// идемпотентной: фоновый прогон может повторно применить события, уже
// применённые синхронной проекцией командной стороны.
type ReadModelWriter interface {
	Apply(ctx context.Context, event eventsourcing.Event) error
}

// NewReadModelProjection строит фоновую проекцию потока событий заказа в
// read model — «догоняющий» механизм на случай падения синхронной проекции.
// Запуск: RunOnce() на старте, Run(ctx, pace) фоном.
func NewReadModelProjection(ctx context.Context, fetcher core.Fetcher, writer ReadModelWriter) *eventsourcing.Projection {
	return eventsourcing.NewProjection(fetcher, func(event eventsourcing.Event) error {
		return writer.Apply(ctx, event)
	})
}
