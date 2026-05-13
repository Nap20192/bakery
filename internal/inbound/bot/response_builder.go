package bot

import (
	"fmt"
	"html"
	"strings"
	"time"

	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
	techcarddomain "bakery/internal/domain/techcard"
	"bakery/internal/pkg/helpers"
)

var responses responseBuilder

type responseBuilder struct{}

func (responseBuilder) Start() string {
	return "<b>orderbot</b>\n\n" +
		"Выберите действие кнопками снизу.\n\n" +
		"<b>Заказ</b>\n" +
		"<code>код название количество</code>\n" +
		"Пример: <code>15647 Сосиска в тесте 5</code>\n" +
		"Дата отдельной строкой: <code>13.05.2026</code>\n" +
		"Заказное: <code>5+5</code>\n" +
		"Количество <code>0</code> удаляет позицию из текущего заказа.\n" +
		"Также можно нажать кнопку <b>Удалить позицию</b> и отправить код."
}

func (responseBuilder) Template(template string) string {
	return "<pre>" + html.EscapeString(template) + "</pre>"
}

func (responseBuilder) OrdersList(orders []orderdomain.Order, departments map[int64]string) string {
	var sb strings.Builder
	sb.WriteString("<b>Последние заказы</b>\n\n")
	for _, order := range orders {
		sb.WriteString(fmt.Sprintf("<b><code>%s</code></b>\n", html.EscapeString(order.Number)))
		if !order.FulfillmentDate.IsZero() {
			sb.WriteString(fmt.Sprintf("Дата выполнения: <code>%s</code>\n", html.EscapeString(order.FulfillmentDate.Format("02.01.2006"))))
		}
		if !order.CreatedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("Создан: <code>%s</code>\n", html.EscapeString(order.CreatedAt.Local().Format("02.01.2006 15:04"))))
		}
		if createdBy := orderCreatedBy(order, ""); createdBy != "" {
			sb.WriteString(fmt.Sprintf("От кого: %s\n", html.EscapeString(createdBy)))
		}
		if from := departmentName(departments, order.FromDepartmentID); from != "" {
			sb.WriteString(fmt.Sprintf("Откуда: %s\n", html.EscapeString(from)))
		}
		if to := departmentName(departments, order.ToDepartmentID); to != "" {
			sb.WriteString(fmt.Sprintf("Куда: %s\n", html.EscapeString(to)))
		}
		sb.WriteString(fmt.Sprintf("Позиций: %d\n\n", len(order.Items)))
	}
	return sb.String()
}

func departmentName(departments map[int64]string, id *int64) string {
	if id == nil {
		return ""
	}
	return strings.TrimSpace(departments[*id])
}

func (responseBuilder) BulkOrderCheck(result orderdomain.BulkOrderValidationResult) string {
	var sb strings.Builder
	sb.WriteString("Проверка заказа\n\n")
	if !result.FulfillmentDate.IsZero() {
		sb.WriteString(fmt.Sprintf("Дата выполнения: <code>%s</code>\n\n", html.EscapeString(result.FulfillmentDate.Format("02.01.2006"))))
	}
	sb.WriteString(fmt.Sprintf("Распознано: %d\n", len(result.ValidItems)))
	for _, item := range result.ValidItems {
		sb.WriteString(fmt.Sprintf(
			"%s %s %s\n",
			html.EscapeString(item.Code),
			html.EscapeString(item.ProductName),
			html.EscapeString(formatOrderItemQuantity(item)),
		))
	}

	if len(result.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("\nОшибки: %d\n", len(result.Errors)))
		writeValidationErrors(&sb, result.Errors)
		sb.WriteString("\nБудут отправлены только корректные позиции.")
	} else {
		sb.WriteString("\nОшибок нет.")
	}
	return sb.String()
}

func (responseBuilder) ValidationErrors(errors []orderdomain.BulkOrderValidationError) string {
	if len(errors) == 0 {
		return "Не удалось распознать позиции в сообщении."
	}

	var sb strings.Builder
	sb.WriteString("Заказ не распознан\n\n")
	writeValidationErrors(&sb, errors)
	sb.WriteString("\nОтправьте новый заказ целиком.")
	return sb.String()
}

func (responseBuilder) OrderSummary(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Заказ <code>%s</code> отправлен</b>\n\n", html.EscapeString(order.Number)))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (responseBuilder) OrderView(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Заказ <code>%s</code></b>\n\n", html.EscapeString(order.Number)))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (responseBuilder) OrderUpdated(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Заказ <code>%s</code> обновлен</b>\n\n", html.EscapeString(order.Number)))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (responseBuilder) OrderDraft(orderNumber string, items []orderdomain.OrderItem, fulfillmentDate time.Time, errors []orderdomain.BulkOrderValidationError) string {
	var sb strings.Builder
	if strings.TrimSpace(orderNumber) != "" {
		sb.WriteString(fmt.Sprintf("<b>Редактирование <code>%s</code></b>\n\n", html.EscapeString(orderNumber)))
	} else {
		sb.WriteString("<b>Текущий заказ</b>\n\n")
	}
	if !fulfillmentDate.IsZero() {
		sb.WriteString(fmt.Sprintf("Дата выполнения: <code>%s</code>\n", html.EscapeString(fulfillmentDate.Format("02.01.2006"))))
	}
	sb.WriteString(fmt.Sprintf("Позиций: %d\n\n", len(items)))
	if len(items) == 0 {
		sb.WriteString("Добавьте позиции сообщением или выберите шаблон через /templates.")
	} else {
		for _, it := range items {
			sb.WriteString(fmt.Sprintf(
				"• <code>%s</code> %s - %s\n",
				html.EscapeString(it.Code),
				html.EscapeString(it.ProductName),
				html.EscapeString(formatOrderItemQuantity(it)),
			))
		}
	}
	if len(errors) > 0 {
		sb.WriteString(fmt.Sprintf("\nОшибки: %d\n", len(errors)))
		writeValidationErrors(&sb, errors)
	}
	return sb.String()
}

func writeOrderDetails(sb *strings.Builder, order orderdomain.Order, fromDepartment string, toDepartment string) {
	if !order.CreatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("Время: <code>%s</code>\n", html.EscapeString(order.CreatedAt.Local().Format("02.01.2006 15:04"))))
	}
	if !order.FulfillmentDate.IsZero() {
		sb.WriteString(fmt.Sprintf("Дата выполнения: <code>%s</code>\n", html.EscapeString(order.FulfillmentDate.Format("02.01.2006"))))
	}
	if createdBy := orderCreatedBy(order, fromDepartment); createdBy != "" {
		sb.WriteString(fmt.Sprintf("От кого: %s\n", html.EscapeString(createdBy)))
	}
	if fromDepartment != "" {
		sb.WriteString(fmt.Sprintf("Откуда: %s\n", html.EscapeString(fromDepartment)))
	}
	if toDepartment != "" {
		sb.WriteString(fmt.Sprintf("Куда: %s\n", html.EscapeString(toDepartment)))
	}
	sb.WriteString("\n<b>Состав заказа:</b>\n")
	for _, it := range order.Items {
		sb.WriteString(fmt.Sprintf(
			"• <code>%s</code> %s - %s\n",
			html.EscapeString(it.Code),
			html.EscapeString(it.ProductName),
			html.EscapeString(formatOrderItemQuantity(it)),
		))
	}
	sb.WriteString(fmt.Sprintf(
		"\nМониторинг: <code>/monitor %s</code>",
		html.EscapeString(order.Number),
	))
}

func orderCreatedBy(order orderdomain.Order, fallback string) string {
	username := strings.TrimSpace(order.CreatedByUsername)
	if username != "" {
		if strings.HasPrefix(username, "@") {
			return username
		}
		return "@" + username
	}
	return strings.TrimSpace(fallback)
}

func formatOrderItemQuantity(item orderdomain.OrderItem) string {
	quantity := helpers.FormatQuantity(item.Quantity)
	if item.ReservedQuantity <= 0 {
		return quantity
	}
	return quantity + "+" + helpers.FormatQuantity(item.ReservedQuantity)
}

func (responseBuilder) MonitorReports(order orderdomain.Order, reports []monitoringdomain.IngredientReport) string {
	var sb strings.Builder
	sb.WriteString("<b>Мониторинг</b>\n\n")
	sb.WriteString(fmt.Sprintf("Заказ: <code>%s</code>\n\n", html.EscapeString(order.Number)))

	for _, report := range reports {
		sb.WriteString(fmt.Sprintf(
			"<b><code>%s</code> %s</b>\n",
			html.EscapeString(report.Ingredient.ProductCode),
			html.EscapeString(report.Ingredient.ProductName),
		))
		sb.WriteString(fmt.Sprintf(
			"Итого: %s %s\n",
			helpers.FormatQuantity(report.Ingredient.Quantity),
			html.EscapeString(report.Ingredient.Unit),
		))

		for _, item := range report.Breakdown {
			sb.WriteString(fmt.Sprintf(
				"• <code>%s</code> %s: %s / %s %s\n",
				html.EscapeString(item.OrderItemCode),
				html.EscapeString(item.OrderItemName),
				helpers.FormatQuantity(item.OrderItemQuantity),
				helpers.FormatQuantity(item.IngredientQuantity),
				html.EscapeString(report.Ingredient.Unit),
			))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (responseBuilder) TechCard(card techcarddomain.TechCard) string {
	var sb strings.Builder
	sb.WriteString("<b>Техкарта</b>\n\n")
	sb.WriteString(fmt.Sprintf("<b><code>%s</code> %s</b>\n", html.EscapeString(card.Code), html.EscapeString(card.Name)))
	if card.Assembly != nil {
		sb.WriteString(fmt.Sprintf("Выход: %s %s\n\n", helpers.FormatQuantity(card.Assembly.AssembledAmount), html.EscapeString(card.Unit)))
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
			sb.WriteString(fmt.Sprintf("line %d: ", errItem.Line))
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
	sb.WriteString(fmt.Sprintf(
		"• <code>%s</code> %s: in %s, middle %s, out %s",
		html.EscapeString(identifier),
		html.EscapeString(product.Name),
		helpers.FormatQuantity(amountIn),
		helpers.FormatQuantity(amountMiddle),
		helpers.FormatQuantity(amountOut),
	))
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
	sb.WriteString(fmt.Sprintf(
		"• <code>%s</code> %s: %s",
		html.EscapeString(identifier),
		html.EscapeString(product.Name),
		helpers.FormatQuantity(amount),
	))
	if product.Unit != "" {
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(product.Unit))
	}
	sb.WriteString("\n")
}
