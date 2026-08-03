package response

import (
	"fmt"
	"html"
	"strings"

	"bakery/internal/pkg/helpers"
	orderdomain "bakery/internal/services/order/domain"
)

func OrderSummary(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>🆕 🆕 🆕 Заказ <code>%s</code> отправлен</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func OrderUpdated(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>🔄 🔄 🔄 Заказ <code>%s</code> обновлён</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	writeOrderChanges(&sb, order.History)
	return sb.String()
}

func OrderCancelled(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>❌ ❌ ❌ Заказ <code>%s</code> отменён</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	if actor := strings.TrimPrefix(strings.TrimSpace(order.CancelledByUsername), "@"); actor != "" {
		fmt.Fprintf(&sb, "\nОтменил: @%s\n", html.EscapeString(actor))
	}
	return sb.String()
}

func OrderRestored(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>♻️ ♻️ ♻️ Заказ <code>%s</code> восстановлен</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

// OrderProduced — уведомление об отработке: сколько реально испечено по
// каждой позиции в сравнении с заявкой.
func OrderProduced(order orderdomain.Order, byUsername string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>🥖 🥖 🥖 Отработка по заказу <code>%s</code></b>\n\n", html.EscapeString(order.Number))
	if actor := strings.TrimPrefix(strings.TrimSpace(byUsername), "@"); actor != "" {
		fmt.Fprintf(&sb, "Внёс: @%s\n", html.EscapeString(actor))
	}
	sb.WriteString("\n<b>Заявка → испечено:</b>\n")
	for _, item := range order.Items {
		if item.ProducedQuantity == nil {
			continue
		}
		ordered := item.ProductionQuantity()
		produced := *item.ProducedQuantity
		marker := ""
		switch {
		case produced < ordered:
			marker = " ⚠️"
		case produced > ordered:
			marker = " ➕"
		}
		fmt.Fprintf(&sb, "%s: %s → %s%s",
			html.EscapeString(item.ProductName),
			helpers.FormatQuantity(ordered),
			helpers.FormatQuantity(produced),
			marker,
		)
		if reason := strings.TrimSpace(item.ProducedReason); reason != "" {
			fmt.Fprintf(&sb, " — %s", html.EscapeString(reason))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// OrderProductionCleared — уведомление о снятии отработки.
func OrderProductionCleared(order orderdomain.Order, byUsername string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>↩️ ↩️ ↩️ Отработка по заказу <code>%s</code> снята</b>\n", html.EscapeString(order.Number))
	if actor := strings.TrimPrefix(strings.TrimSpace(byUsername), "@"); actor != "" {
		fmt.Fprintf(&sb, "Снял: @%s\n", html.EscapeString(actor))
	}
	return sb.String()
}

// writeOrderChanges renders the latest set of position changes (added /
// updated / removed) so the workshop sees what changed in this update.
func writeOrderChanges(sb *strings.Builder, history []orderdomain.OrderHistory) {
	if len(history) == 0 || len(history[0].Items) == 0 {
		return
	}
	if editor := strings.TrimPrefix(strings.TrimSpace(history[0].ChangedByUsername), "@"); editor != "" {
		fmt.Fprintf(sb, "\n<b>Изменения</b> (@%s):\n", html.EscapeString(editor))
	} else {
		sb.WriteString("\n<b>Изменения:</b>\n")
	}
	for _, item := range history[0].Items {
		fmt.Fprintf(
			sb, "%s %s %s\n",
			changeLabel(item.ChangeType),
			html.EscapeString(item.ProductName),
			changeQuantityText(item),
		)
	}
	sb.WriteString("\n")
}

func changeLabel(changeType string) string {
	switch changeType {
	case "added":
		return "[добавлено]"
	case "updated":
		return "[изменено]"
	case "removed":
		return "[удалено]"
	default:
		return "[обновлено]"
	}
}

func changeQuantityText(item orderdomain.OrderHistoryItem) string {
	switch item.ChangeType {
	case "updated":
		return historyQuantity(item.OldQuantity, item.OldReservedQuantity) + " → " + historyQuantity(item.NewQuantity, item.NewReservedQuantity)
	case "removed":
		return "было " + historyQuantity(item.OldQuantity, item.OldReservedQuantity)
	default:
		return historyQuantity(item.NewQuantity, item.NewReservedQuantity)
	}
}

func historyQuantity(quantity, reserved *float64) string {
	base := helpers.FormatQuantity(derefQuantity(quantity))
	if r := derefQuantity(reserved); r > 0 {
		return base + "+" + helpers.FormatQuantity(r)
	}
	return base
}

func derefQuantity(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func writeOrderDetails(sb *strings.Builder, order orderdomain.Order, fromDepartment string, toDepartment string) {
	if order.Category != nil && strings.TrimSpace(order.Category.Name) != "" {
		fmt.Fprintf(sb, "Тип: %s\n", html.EscapeString(order.Category.Name))
	}
	if !order.FulfillmentDate.IsZero() {
		fmt.Fprintf(sb, "Выполнить: <code>%s</code>\n", html.EscapeString(order.FulfillmentDate.Format("02.01.2006")))
	}
	if fromDepartment != "" {
		fmt.Fprintf(sb, "Откуда: %s\n", html.EscapeString(fromDepartment))
	}
	if toDepartment != "" {
		fmt.Fprintf(sb, "Куда: %s\n", html.EscapeString(toDepartment))
	}
	if sender := strings.TrimPrefix(strings.TrimSpace(order.CreatedByUsername), "@"); sender != "" {
		fmt.Fprintf(sb, "Отправитель: @%s\n", html.EscapeString(sender))
	}
	sb.WriteString("\n<b>Состав:</b>\n")
	writeOrderItemsCodeBlock(sb, order.Items)
	writeOrderComments(sb, order.Comments)
}

func writeOrderComments(sb *strings.Builder, comments orderdomain.OrderComments) {
	if len(comments.Items) > 0 {
		sb.WriteString("\n<b>Комментарии:</b>\n")
		for _, c := range comments.Items {
			fmt.Fprintf(sb, "• %s — %s\n", html.EscapeString(c.ProductName), html.EscapeString(c.Comment))
		}
	}
	if strings.TrimSpace(comments.General) != "" {
		fmt.Fprintf(sb, "\nКомментарий: %s\n", html.EscapeString(comments.General))
	}
}

func writeOrderItemsCodeBlock(sb *strings.Builder, items []orderdomain.OrderItem) {
	for _, item := range items {
		fmt.Fprintf(sb, "%s %s\n", html.EscapeString(item.ProductName), html.EscapeString(formatOrderItemQuantity(item)))
	}
}

func formatOrderItemQuantity(item orderdomain.OrderItem) string {
	quantity := helpers.FormatQuantity(item.Quantity)
	if item.ReservedQuantity <= 0 {
		return quantity
	}
	return quantity + "+" + helpers.FormatQuantity(item.ReservedQuantity)
}
