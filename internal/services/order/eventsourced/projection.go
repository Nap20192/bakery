package eventsourced

import (
	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
)

// ReadModelWriter — порт проекции: применяет события заказа к read model
// (существующие таблицы orders/order_items). Реализация — этап 2 перевода
// (адаптер поверх sqlc в infra/repo); интерфейс объявлен у потребителя.
type ReadModelWriter interface {
	ApplyCreated(number string, data Created) error
	ApplyItemsUpdated(number string, data ItemsUpdated) error
	ApplyCancelled(number string, data Cancelled) error
	ApplyRestored(number string, data Restored) error
	ApplyProductionRecorded(number string, data ProductionRecorded) error
	ApplyProductionCleared(number string, data ProductionCleared) error
}

// NewReadModelProjection строит проекцию потока событий заказа в read model.
// Запуск: RunOnce() на старте (догнать хвост после падений синхронной
// проекции) + Run(ctx, pace) фоном.
func NewReadModelProjection(fetcher core.Fetcher, writer ReadModelWriter) *eventsourcing.Projection {
	return eventsourcing.NewProjection(fetcher, func(event eventsourcing.Event) error {
		return applyEvent(writer, event)
	})
}

// applyEvent — общий диспетчер «событие → read model»: используется и
// синхронной проекцией командной стороны, и фоновым прогоном.
func applyEvent(writer ReadModelWriter, event eventsourcing.Event) error {
	number := event.AggregateID()
	switch data := event.Data().(type) {
	case *Created:
		return writer.ApplyCreated(number, *data)
	case *ItemsUpdated:
		return writer.ApplyItemsUpdated(number, *data)
	case *Cancelled:
		return writer.ApplyCancelled(number, *data)
	case *Restored:
		return writer.ApplyRestored(number, *data)
	case *ProductionRecorded:
		return writer.ApplyProductionRecorded(number, *data)
	case *ProductionCleared:
		return writer.ApplyProductionCleared(number, *data)
	}
	return nil
}
