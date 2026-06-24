package response

import (
	"fmt"
	"html"
	"strings"
	"time"

	"bakery/internal/pkg/helpers"
	monitoringdomain "bakery/internal/services/monitor/domain"
	orderdomain "bakery/internal/services/order/domain"
	techcarddomain "bakery/internal/services/techcard/domain"
)

const ordersWebURL = "https://orders-production-3e6e.up.railway.app/"

type Builder struct{}

func (Builder) Start() string {
	return "<b>orderbot</b>\n\n" +
		"Магазин — позиции сообщением. Цех — /orders.\n\n" +
		ordersWebURL
}

func (Builder) Help() string {
	return "<b>orderbot</b>\n\n" +
		"Заказ — позиции сообщением:\n" +
		"<code>Сосиска в тесте 5</code>\n" +
		"<code>Сосиска в тесте 5+2</code>\n" +
		"<code>Кокрок 5 // комментарий</code>\n" +
		"<code>25.05.2026</code>\n\n" +
		"/orders /templates /cancel\n\n" +
		ordersWebURL
}

func (Builder) Template(template string) string {
	return "<pre>" + html.EscapeString(template) + "</pre>"
}

func (Builder) ValidationErrors(errors []orderdomain.BulkOrderValidationError) string {
	if len(errors) == 0 {
		return "Не удалось распознать позиции в сообщении."
	}

	var sb strings.Builder
	sb.WriteString("Заказ не распознан\n\n")
	writeValidationErrors(&sb, errors)
	return sb.String()
}

func (Builder) OrderSummary(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Заказ <code>%s</code> отправлен</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (Builder) OrderView(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Заказ <code>%s</code></b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (Builder) OrderCopy(order orderdomain.Order) string {
	var sb strings.Builder
	sb.WriteString("<pre>")
	if !order.FulfillmentDate.IsZero() {
		sb.WriteString(html.EscapeString(order.FulfillmentDate.Format("02.01.2006")))
		sb.WriteString("\n")
	}
	for _, item := range order.Items {
		sb.WriteString(html.EscapeString(item.ProductName))
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(formatOrderItemQuantity(item)))
		sb.WriteString("\n")
	}
	sb.WriteString("</pre>")
	return sb.String()
}

func (Builder) OrderUpdated(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Заказ <code>%s</code> обновлен</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	writeOrderChanges(&sb, order.History)
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
		fmt.Fprintf(sb, "%s %s %s\n",
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

func (Builder) OrderDraft(orderNumber string, items []orderdomain.OrderItem, fulfillmentDate time.Time, errors []orderdomain.BulkOrderValidationError) string {
	var sb strings.Builder
	if strings.TrimSpace(orderNumber) != "" {
		fmt.Fprintf(&sb, "<b>Редактирование <code>%s</code></b>\n\n", html.EscapeString(orderNumber))
	} else {
		sb.WriteString("<b>Текущий заказ</b>\n\n")
	}
	if !fulfillmentDate.IsZero() {
		fmt.Fprintf(&sb, "Выполнить: <code>%s</code>\n", html.EscapeString(fulfillmentDate.Format("02.01.2006")))
	}
	fmt.Fprintf(&sb, "Позиций: %d\n\n", len(items))
	if len(items) == 0 {
		sb.WriteString("Добавьте позиции сообщением.")
	} else {
		writeOrderItemsCodeBlock(&sb, items)
		writeOrderComments(&sb, orderdomain.CommentsFromItems(items))
	}
	if len(errors) > 0 {
		fmt.Fprintf(&sb, "\nОшибки: %d\n", len(errors))
		writeValidationErrors(&sb, errors)
	}
	return sb.String()
}

func writeOrderDetails(sb *strings.Builder, order orderdomain.Order, fromDepartment string, toDepartment string) {
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
	sb.WriteString("<pre>")
	for _, item := range items {
		fmt.Fprintf(sb, "%s %s\n", html.EscapeString(item.ProductName), html.EscapeString(formatOrderItemQuantity(item)))
	}
	sb.WriteString("</pre>\n")
}

func formatOrderItemQuantity(item orderdomain.OrderItem) string {
	quantity := helpers.FormatQuantity(item.Quantity)
	if item.ReservedQuantity <= 0 {
		return quantity
	}
	return quantity + "+" + helpers.FormatQuantity(item.ReservedQuantity)
}

func (Builder) MonitorReports(order orderdomain.Order, reports []monitoringdomain.IngredientReport) string {
	var sb strings.Builder
	sb.WriteString("<b>Калькуляция</b>\n\n")
	fmt.Fprintf(&sb, "Заказ: <code>%s</code>\n\n", html.EscapeString(order.Number))

	for _, report := range reports {
		fmt.Fprintf(&sb, "<b>%s</b>\n",
			html.EscapeString(report.Ingredient.ProductName))
		fmt.Fprintf(&sb, "Итого: %s %s\n",
			helpers.FormatQuantity(report.Ingredient.Quantity),
			html.EscapeString(report.Ingredient.Unit))

		for _, item := range report.Breakdown {
			fmt.Fprintf(&sb, "• %s: %s / %s %s\n",
				html.EscapeString(item.OrderItemName),
				helpers.FormatQuantity(item.OrderItemQuantity),
				helpers.FormatQuantity(item.IngredientQuantity),
				html.EscapeString(report.Ingredient.Unit))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (Builder) BatchMonitorReports(report monitoringdomain.BatchMonitoringReport) string {
	var sb strings.Builder
	sb.WriteString("<b>Калькуляция выбранных заказов</b>\n\n")
	if len(report.Orders) > 0 {
		sb.WriteString("<b>Заказы:</b>\n")
		for _, order := range report.Orders {
			fmt.Fprintf(&sb, "• <code>%s</code>\n", html.EscapeString(order.OrderNumber))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("<b>Итого по всем заказам</b>\n\n")
	writeMonitorReports(&sb, report.TotalReports, false)

	for _, order := range report.Orders {
		fmt.Fprintf(&sb, "<b>Заказ <code>%s</code></b>\n\n", html.EscapeString(order.OrderNumber))
		writeMonitorReports(&sb, order.Reports, true)
	}

	return sb.String()
}

func writeMonitorReports(sb *strings.Builder, reports []monitoringdomain.IngredientReport, totalsOnly bool) {
	for _, report := range reports {
		fmt.Fprintf(sb,
			"<b>%s</b>\n",
			html.EscapeString(report.Ingredient.ProductName))
		fmt.Fprintf(sb,
			"Итого: %s %s\n",
			helpers.FormatQuantity(report.Ingredient.Quantity),
			html.EscapeString(report.Ingredient.Unit))

		if !totalsOnly {
			for _, item := range report.Breakdown {
				fmt.Fprintf(sb,
					"• %s: %s / %s %s\n",
					html.EscapeString(item.OrderItemName),
					helpers.FormatQuantity(item.OrderItemQuantity),
					helpers.FormatQuantity(item.IngredientQuantity),
					html.EscapeString(report.Ingredient.Unit))
			}
		}
		sb.WriteString("\n")
	}
}

func (Builder) TechCard(card techcarddomain.TechCard) string {
	var sb strings.Builder
	sb.WriteString("<b>Техкарта</b>\n\n")
	fmt.Fprintf(&sb, "<b><code>%s</code> %s</b>\n", html.EscapeString(card.Code), html.EscapeString(card.Name))
	if card.Assembly != nil {
		fmt.Fprintf(&sb, "Выход: %s %s\n\n", helpers.FormatQuantity(card.Assembly.AssembledAmount), html.EscapeString(card.Unit))
		for _, item := range card.Assembly.Items {
			product := card.Products[item.ProductID]
			writeTechCardItem(&sb, product, item.AmountIn, item.AmountMiddle, item.AmountOut)
		}
		return sb.String()
	}
	if card.Prepared != nil {
		sb.WriteString("\n")
		for _, item := range card.Prepared.Items {
			product := card.Products[item.ProductID]
			writePreparedTechCardItem(&sb, product, item.Amount)
		}
		return sb.String()
	}
	sb.WriteString("Техкарта не найдена.")
	return sb.String()
}

func writeValidationErrors(sb *strings.Builder, errors []orderdomain.BulkOrderValidationError) {
	for _, errItem := range errors {
		if errItem.Line > 0 {
			fmt.Fprintf(sb, "строка %d: ", errItem.Line)
		}
		if errItem.Raw != "" {
			sb.WriteString("\"")
			sb.WriteString(html.EscapeString(errItem.Raw))
			sb.WriteString("\" ")
		}
		if errItem.Code != "" {
			sb.WriteString(html.EscapeString(errItem.Code))
			sb.WriteString(" ")
		}
		if errItem.Name != "" {
			sb.WriteString(html.EscapeString(errItem.Name))
			sb.WriteString(" ")
		}
		sb.WriteString("- ")
		sb.WriteString(html.EscapeString(errItem.Message))
		sb.WriteString("\n")
	}
}

func writeTechCardItem(sb *strings.Builder, product techcarddomain.TechCardProduct, amountIn, amountMiddle, amountOut float64) {
	identifier := product.Code
	if identifier == "" {
		identifier = product.ID
	}
	fmt.Fprintf(sb,
		"• <code>%s</code> %s: in %s, middle %s, out %s",
		html.EscapeString(identifier),
		html.EscapeString(product.Name),
		helpers.FormatQuantity(amountIn),
		helpers.FormatQuantity(amountMiddle),
		helpers.FormatQuantity(amountOut))
	if product.Unit != "" {
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(product.Unit))
	}
	sb.WriteString("\n")
}

func writePreparedTechCardItem(sb *strings.Builder, product techcarddomain.TechCardProduct, amount float64) {
	identifier := product.Code
	if identifier == "" {
		identifier = product.ID
	}
	fmt.Fprintf(sb,
		"• <code>%s</code> %s: %s",
		html.EscapeString(identifier),
		html.EscapeString(product.Name),
		helpers.FormatQuantity(amount))
	if product.Unit != "" {
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(product.Unit))
	}
	sb.WriteString("\n")
}
