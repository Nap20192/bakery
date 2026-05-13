package bot

import (
	"log/slog"
	"time"

	orderdomain "bakery/internal/domain/order"
)

const (
	sessionTTL      = 12 * time.Hour // незавершённая заявка живёт 12 часов
	cleanupInterval = time.Hour      // проверяем каждый час
)

type session struct {
	items            []orderdomain.OrderItem
	fromDepartmentID *int64
	toDepartmentID   *int64
	fulfillmentDate  time.Time
	editOrderNumber  string
	waitingTemplate  bool
	waitingDelete    bool
	updatedAt        time.Time // время последнего изменения
}

// updateSession атомарно изменяет сессию пользователя под мьютексом.
func (b *OrderBot) updateSession(uid int64, fn func(*session)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.sessions[uid]
	if !ok {
		s = &session{}
		b.sessions[uid] = s
	}
	fn(s)
	s.updatedAt = time.Now()
}

func (b *OrderBot) clearSession(uid int64) {
	b.updateSession(uid, func(s *session) {
		s.items = nil
		s.fulfillmentDate = time.Time{}
		s.editOrderNumber = ""
		s.waitingTemplate = false
		s.waitingDelete = false
	})
}

func (b *OrderBot) isWaitingDelete(uid int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[uid]
	return s != nil && s.waitingDelete
}

func mergeSessionItems(existing []orderdomain.OrderItem, incoming []orderdomain.OrderItem) []orderdomain.OrderItem {
	merged := make([]orderdomain.OrderItem, 0, len(existing)+len(incoming))
	index := make(map[string]int, len(existing)+len(incoming))
	for _, item := range existing {
		if item.ProductionQuantity() <= 0 {
			continue
		}
		index[item.Code] = len(merged)
		merged = append(merged, item)
	}
	for _, item := range incoming {
		if item.ProductionQuantity() <= 0 {
			if idx, ok := index[item.Code]; ok {
				merged = append(merged[:idx], merged[idx+1:]...)
				index = make(map[string]int, len(merged))
				for i, existingItem := range merged {
					index[existingItem.Code] = i
				}
			}
			continue
		}
		if idx, ok := index[item.Code]; ok {
			merged[idx].ProductName = item.ProductName
			merged[idx].Quantity = item.Quantity
			merged[idx].ReservedQuantity = item.ReservedQuantity
			continue
		}
		index[item.Code] = len(merged)
		merged = append(merged, item)
	}
	return merged
}

// cleanupLoop удаляет сессии, которые не обновлялись дольше sessionTTL.
func (b *OrderBot) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		var expired []int64
		for uid, s := range b.sessions {
			if !s.updatedAt.IsZero() && now.Sub(s.updatedAt) > sessionTTL {
				expired = append(expired, uid)
			}
		}
		for _, uid := range expired {
			delete(b.sessions, uid)
		}
		b.mu.Unlock()
		if len(expired) > 0 {
			slog.Info("session cleanup", "expired", len(expired))
		}
	}
}
