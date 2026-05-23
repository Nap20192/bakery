package bot

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
	techcarddomain "bakery/internal/domain/techcard"
	"bakery/internal/pkg/helpers"
)

var responses responseBuilder

const ordersWebURL = "https://orders-production-3e6e.up.railway.app/"

type responseBuilder struct{}

func (responseBuilder) Start() string {
	return "<b>orderbot</b>\n\n" +
		"Выберите действие кнопками снизу.\n" +
		fmt.Sprintf("Все заказы: %s\n", ordersWebURL) +
		"Инструкция: /help"
}

func (responseBuilder) Help() string {
	return "<b>Как пользоваться orderbot</b>\n\n" +
		"<b>Сначала выберите роль</b>\n" +
		"Нажмите <b>Выбрать магазин</b>, если вы оформляете заказ от магазина.\n" +
		"Нажмите <b>Выбрать цех</b>, если вы смотрите заказы и считаете тесто.\n\n" +
		"<b>Магазин: создать заказ</b>\n" +
		"1. Нажмите <b>Шаблоны</b> и выберите нужную группу.\n" +
		"2. В шаблоне замените <code>0</code> на нужное количество.\n" +
		"3. Отправьте готовый текст боту одним сообщением.\n" +
		"4. Проверьте <b>Текущий заказ</b> и нажмите <b>Отправить заказ</b>.\n\n" +
		"Можно писать без шаблона:\n" +
		"<code>Сосиска в тесте 5</code>\n" +
		"<code>Кокрок с картофелем 12</code>\n\n" +
		"Дата выполнения пишется отдельной строкой:\n" +
		"<code>25.05.2026</code>\n" +
		"Если дату не указать, бот поставит следующий день.\n\n" +
		"Заказное количество пишется через плюс:\n" +
		"<code>Сосиска в тесте 5+2</code>\n" +
		"Здесь <code>5</code> обычное количество, <code>2</code> заказное.\n\n" +
		"Если отправить блюдо повторно, количество обновится.\n" +
		"Чтобы удалить блюдо из текущего заказа, отправьте его с количеством <code>0</code>.\n\n" +
		"<b>Магазин: смотреть заказы</b>\n" +
		"Нажмите <b>Последние заказы</b>. Магазин видит только свои последние заказы.\n" +
		"В заказе можно открыть состав, скопировать текст, изменить заказ или посчитать тесто.\n\n" +
		"<b>Цех: смотреть и считать</b>\n" +
		"Нажмите <b>Последние заказы</b>. Цех видит заказы всех магазинов.\n" +
		"Выберите фильтр по магазину или дате кнопками.\n" +
		"После фильтрации можно открыть заказ или нажать <b>Посчитать тесто по заказам</b>.\n" +
		"Расчёт идёт по всем заказам, которые сейчас отфильтрованы.\n\n" +
		"<b>Полезные команды</b>\n" +
		"/start - выбрать магазин или цех заново.\n" +
		"/templates - открыть шаблоны.\n" +
		"/orders - последние заказы.\n" +
		"/cancel - отменить текущий несохранённый заказ.\n" +
		fmt.Sprintf("\nВеб-просмотр: %s", ordersWebURL)
}

func (responseBuilder) Template(template string) string {
	return "<pre>" + html.EscapeString(template) + "</pre>"
}

func (responseBuilder) BulkOrderCheck(result orderdomain.BulkOrderValidationResult) string {
	var sb strings.Builder
	sb.WriteString("Проверка заказа\n\n")
	if !result.FulfillmentDate.IsZero() {
		fmt.Fprintf(&sb, "Дата выполнения: <code>%s</code>\n\n", html.EscapeString(result.FulfillmentDate.Format("02.01.2006")))
	}
	fmt.Fprintf(&sb, "Распознано: %d\n", len(result.ValidItems))
	for _, item := range result.ValidItems {
		fmt.Fprintf(
			&sb,
			"%s %s %s\n",
			html.EscapeString(item.Code),
			html.EscapeString(item.ProductName),
			html.EscapeString(formatOrderItemQuantity(item)),
		)
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(&sb, "\nОшибки: %d\n", len(result.Errors))
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
	fmt.Fprintf(&sb, "<b>Заказ <code>%s</code> отправлен</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (responseBuilder) OrderView(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Заказ <code>%s</code></b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (responseBuilder) OrderCopy(order orderdomain.Order) string {
	var sb strings.Builder
	sb.WriteString("<pre>")
	if !order.FulfillmentDate.IsZero() {
		sb.WriteString(html.EscapeString(order.FulfillmentDate.Format("02.01.2006")))
		sb.WriteString("\n")
	}
	for _, item := range order.Items {
		sb.WriteString(html.EscapeString(item.Code))
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(item.ProductName))
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(formatOrderItemQuantity(item)))
		sb.WriteString("\n")
	}
	sb.WriteString("</pre>")
	return sb.String()
}

func (responseBuilder) OrderUpdated(order orderdomain.Order, fromDepartment string, toDepartment string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Заказ <code>%s</code> обновлен</b>\n\n", html.EscapeString(order.Number))
	writeOrderDetails(&sb, order, fromDepartment, toDepartment)
	return sb.String()
}

func (responseBuilder) OrderDraft(orderNumber string, items []orderdomain.OrderItem, fulfillmentDate time.Time, errors []orderdomain.BulkOrderValidationError) string {
	var sb strings.Builder
	if strings.TrimSpace(orderNumber) != "" {
		fmt.Fprintf(&sb, "<b>Редактирование <code>%s</code></b>\n\n", html.EscapeString(orderNumber))
	} else {
		sb.WriteString("<b>Текущий заказ</b>\n\n")
	}
	if !fulfillmentDate.IsZero() {
		fmt.Fprintf(&sb, "Дата выполнения: <code>%s</code>\n", html.EscapeString(fulfillmentDate.Format("02.01.2006")))
	}
	fmt.Fprintf(&sb, "Позиций: %d\n\n", len(items))
	if len(items) == 0 {
		sb.WriteString("Добавьте позиции сообщением или выберите шаблон через /templates.")
	} else {
		for _, it := range items {
			sb.WriteString(fmt.Sprintf(
				"<code>%s</code> %s %s\n",
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
		sb.WriteString(fmt.Sprintf("<b>Откуда: %s</b>\n", html.EscapeString(fromDepartment)))
	}
	if toDepartment != "" {
		sb.WriteString(fmt.Sprintf("Куда: %s\n", html.EscapeString(toDepartment)))
	}
	sb.WriteString("\n<b>Состав заказа:</b>\n")
	latestChanges := latestOrderChanges(order)
	for _, it := range order.Items {
		mark := ""
		if changeType := latestChanges[it.Code]; changeType != "" {
			mark = " " + orderChangeLabel(changeType)
		}
		sb.WriteString(fmt.Sprintf(
			"<code>%s</code> %s %s%s\n",
			html.EscapeString(it.Code),
			html.EscapeString(it.ProductName),
			html.EscapeString(formatOrderItemQuantity(it)),
			mark,
		))
	}
	writeOrderHistorySummary(sb, order)
	sb.WriteString(fmt.Sprintf(
		"\nКалькуляция: <code>/monitor %s</code>",
		html.EscapeString(order.Number),
	))
	sb.WriteString(fmt.Sprintf("\nОткрыть заказ: %s", html.EscapeString(orderWebURL(order.Number))))
}

func latestOrderChanges(order orderdomain.Order) map[string]string {
	result := make(map[string]string)
	if len(order.History) == 0 {
		return result
	}
	for _, item := range order.History[0].Items {
		if strings.TrimSpace(item.ProductCode) == "" {
			continue
		}
		result[item.ProductCode] = item.ChangeType
	}
	return result
}

func writeOrderHistorySummary(sb *strings.Builder, order orderdomain.Order) {
	if len(order.History) == 0 {
		return
	}
	sb.WriteString("\n<b>История изменений:</b>\n")
	for _, history := range order.History {
		if !history.ChangedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("<code>%s</code>", html.EscapeString(history.ChangedAt.Local().Format("02.01.2006 15:04"))))
		}
		if strings.TrimSpace(history.ChangedByUsername) != "" {
			sb.WriteString(fmt.Sprintf(" %s", html.EscapeString(orderCreatedBy(orderdomain.Order{CreatedByUsername: history.ChangedByUsername}, ""))))
		}
		sb.WriteByte('\n')
		for _, item := range history.Items {
			writeOrderHistoryItem(sb, item)
		}
	}
}

func writeOrderHistoryItem(sb *strings.Builder, item orderdomain.OrderHistoryItem) {
	sb.WriteString(fmt.Sprintf(
		"%s <code>%s</code> %s",
		orderChangeLabel(item.ChangeType),
		html.EscapeString(item.ProductCode),
		html.EscapeString(item.ProductName),
	))
	if item.ChangeType == "updated" {
		sb.WriteString(fmt.Sprintf(": %s → %s", historyQuantity(item.OldQuantity, item.OldReservedQuantity), historyQuantity(item.NewQuantity, item.NewReservedQuantity)))
	}
	if item.ChangeType == "added" {
		sb.WriteString(fmt.Sprintf(": %s", historyQuantity(item.NewQuantity, item.NewReservedQuantity)))
	}
	if item.ChangeType == "removed" {
		sb.WriteString(fmt.Sprintf(": было %s", historyQuantity(item.OldQuantity, item.OldReservedQuantity)))
	}
	sb.WriteByte('\n')
}

func historyQuantity(quantity *float64, reserved *float64) string {
	if quantity == nil {
		return "0"
	}
	item := orderdomain.OrderItem{Quantity: *quantity}
	if reserved != nil {
		item.ReservedQuantity = *reserved
	}
	return html.EscapeString(formatOrderItemQuantity(item))
}

func orderChangeLabel(changeType string) string {
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

func orderWebURL(orderNumber string) string {
	base, err := url.Parse(ordersWebURL)
	if err != nil {
		return ordersWebURL
	}
	query := base.Query()
	query.Set("order", strings.TrimSpace(orderNumber))
	base.RawQuery = query.Encode()
	return base.String()
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
	sb.WriteString("<b>Калькуляция</b>\n\n")
	sb.WriteString(fmt.Sprintf("Заказ: <code>%s</code>\n\n", html.EscapeString(order.Number)))

	for _, report := range reports {
		sb.WriteString(fmt.Sprintf(
			"<b>%s</b>\n",
			html.EscapeString(report.Ingredient.ProductName),
		))
		sb.WriteString(fmt.Sprintf(
			"Итого: %s %s\n",
			helpers.FormatQuantity(report.Ingredient.Quantity),
			html.EscapeString(report.Ingredient.Unit),
		))

		for _, item := range report.Breakdown {
			sb.WriteString(fmt.Sprintf(
				"• %s: %s / %s %s\n",
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

func (responseBuilder) BatchMonitorReports(report monitoringdomain.BatchMonitoringReport) string {
	var sb strings.Builder
	sb.WriteString("<b>Калькуляция выбранных заказов</b>\n\n")
	if len(report.Orders) > 0 {
		sb.WriteString("<b>Заказы:</b>\n")
		for _, order := range report.Orders {
			sb.WriteString(fmt.Sprintf("• <code>%s</code>\n", html.EscapeString(order.OrderNumber)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("<b>Итого по всем заказам</b>\n\n")
	writeMonitorReports(&sb, report.TotalReports, false)

	for _, order := range report.Orders {
		sb.WriteString(fmt.Sprintf("<b>Заказ <code>%s</code></b>\n\n", html.EscapeString(order.OrderNumber)))
		writeMonitorReports(&sb, order.Reports, true)
	}

	return sb.String()
}

func writeMonitorReports(sb *strings.Builder, reports []monitoringdomain.IngredientReport, totalsOnly bool) {
	for _, report := range reports {
		sb.WriteString(fmt.Sprintf(
			"<b>%s</b>\n",
			html.EscapeString(report.Ingredient.ProductName),
		))
		sb.WriteString(fmt.Sprintf(
			"Итого: %s %s\n",
			helpers.FormatQuantity(report.Ingredient.Quantity),
			html.EscapeString(report.Ingredient.Unit),
		))

		if !totalsOnly {
			for _, item := range report.Breakdown {
				sb.WriteString(fmt.Sprintf(
					"• %s: %s / %s %s\n",
					html.EscapeString(item.OrderItemName),
					helpers.FormatQuantity(item.OrderItemQuantity),
					helpers.FormatQuantity(item.IngredientQuantity),
					html.EscapeString(report.Ingredient.Unit),
				))
			}
		}
		sb.WriteString("\n")
	}
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
