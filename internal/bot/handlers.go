package bot

import (
	"fmt"
	"html"
	"log"
	"sort"
	"strings"

	"bakery/internal/domain"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleStart(c tele.Context) error {
	return c.Send(
		"🍞 *orderbot — batch-заявки*\n\n"+
			"Отправьте заявку одним сообщением в формате:\n\n"+
			"`Локация`\n"+
			"`Категория`\n"+
			"`Продукт количество`\n\n"+
			"/template — отправить стандартный шаблон\n"+
			"/cancel — отменить неподтверждённую заявку",
		tele.ModeMarkdown,
	)
}

func (b *OrderBot) handleTemplate(c tele.Context) error {
	return c.Send("<pre>"+html.EscapeString(defaultOrderTemplate)+"</pre>", tele.ModeHTML)
}

func (b *AdminBot) handleStart(c tele.Context) error {
	return c.Send(
		"📦 *adminbot — просмотр заказов*\n\n"+
			"/orders — последние заказы\n"+
			"Отправьте один или несколько номеров заказов, чтобы увидеть расчёт теста.",
		tele.ModeMarkdown,
	)
}

func (b *OrderBot) handleCancel(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	return c.Send("Заявка отменена.")
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())

	if isBulkOrder(text) {
		return b.handleBulkOrder(c, text)
	}
	if len(splitNumbers(text)) > 0 {
		return c.Send("Этот бот создаёт batch-заявки. Для просмотра заказов используйте adminbot.")
	}
	if strings.HasPrefix(text, "/") {
		return c.Send("Неизвестная команда.\n\n/start — формат batch-заявки")
	}
	return c.Send("Отправьте batch-заявку одним сообщением. /start — формат заявки.")
}

func (b *AdminBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if len(splitNumbers(text)) > 0 {
		return b.handleOrderLookup(c, text)
	}
	if strings.HasPrefix(text, "/") {
		return c.Send("Неизвестная команда.\n\n/start — показать меню")
	}
	return c.Send("Отправьте номер заказа (например `15042026_ORDER_0001`) или /orders.", tele.ModeMarkdown)
}

func (b *AdminBot) handleOrderLookup(c tele.Context, text string) error {
	numbers := splitNumbers(text)

	if len(numbers) == 1 {
		return b.showSingleOrder(c, numbers[0])
	}
	return b.showMultipleOrders(c, numbers)
}

func (b *AdminBot) showSingleOrder(c tele.Context, number string) error {
	order, err := b.orderRepo.GetByNumber(number)
	if err != nil {
		return c.Send(fmt.Sprintf("Заказ %q не найден.", number))
	}
	dough, err := b.orderSvc.CalculateDough(order, b.doughRepo)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка расчёта теста: %v", err))
	}
	return c.Send(formatOrderSummary(order, dough), tele.ModeMarkdown)
}

func (b *AdminBot) showMultipleOrders(c tele.Context, numbers []string) error {
	var (
		found    []domain.Order
		notFound []string
	)

	for _, num := range numbers {
		order, err := b.orderRepo.GetByNumber(num)
		if err != nil {
			notFound = append(notFound, num)
			continue
		}
		found = append(found, order)
	}

	if len(found) == 0 {
		return c.Send("Ни один из указанных заказов не найден.")
	}

	combined := b.orderSvc.CombineOrders(found)
	dough, err := b.orderSvc.CalculateDough(combined, b.doughRepo)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка расчёта теста: %v", err))
	}

	return c.Send(formatMultiOrderSummary(found, notFound, dough), tele.ModeMarkdown)
}

func splitNumbers(text string) []string {
	var result []string
	for _, part := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ','
	}) {
		if strings.Contains(part, "_ORDER_") {
			result = append(result, part)
		}
	}
	return result
}

func (b *OrderBot) handleConfirm(c tele.Context) error {
	var location string
	var items []domain.OrderItem
	b.updateSession(c.Sender().ID, func(s *session) {
		location = s.location
		items = make([]domain.OrderItem, len(s.items))
		copy(items, s.items)
		s.location = ""
		s.items = nil
	})

	_ = c.Respond()
	if len(items) == 0 {
		return c.Send("Заявка пустая.")
	}

	order, err := b.orderRepo.Create(domain.CreateOrderInput{
		Items:    items,
		Location: location,
	})
	if err != nil {
		log.Printf("ERR create order uid=%d: %v", c.Sender().ID, err)
		return c.Send(fmt.Sprintf("Ошибка создания заказа: %v", err))
	}
	log.Printf("ORDER created %s uid=%d items=%d", order.Number, c.Sender().ID, len(items))

	doughSummary, err := b.orderSvc.CalculateDough(order, b.doughRepo)
	if err != nil {
		log.Printf("ERR dough calc order=%s: %v", order.Number, err)
		return c.Send(fmt.Sprintf("Заказ %s создан, ошибка расчёта теста: %v", order.Number, err))
	}
	for _, d := range doughSummary {
		log.Printf("DOUGH order=%s %s=%.3f кг", order.Number, d.DoughName, d.TotalKg)
	}

	return c.Send(formatOrderSummary(order, doughSummary), tele.ModeMarkdown)
}

func (b *OrderBot) handleCancelCallback(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	_ = c.Respond()
	return c.Send("Заявка отменена.")
}

func (b *AdminBot) handleOrders(c tele.Context) error {
	orders, err := b.orderRepo.List(10)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка: %v", err))
	}
	if len(orders) == 0 {
		return c.Send("Заказов пока нет.")
	}

	var sb strings.Builder
	sb.WriteString("*Последние заказы:*\n\n")
	for _, o := range orders {
		sb.WriteString(fmt.Sprintf("📦 `%s` — %s\n",
			o.Number, o.CreatedAt.Local().Format("02.01.2006 15:04"),
		))
		for _, it := range o.Items {
			sb.WriteString(fmt.Sprintf("  • %s × %d\n", it.Product, it.Quantity))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("_Отправьте номер заказа чтобы увидеть расчёт теста._")
	return c.Send(sb.String(), tele.ModeMarkdown)
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	location, _, parsed := parseBulkOrder(text)

	if len(parsed) == 0 {
		return c.Send("Не удалось распознать позиции в сообщении.")
	}

	catalog := make(map[string]string, len(b.productNames))
	for _, name := range b.productNames {
		catalog[strings.ToLower(name)] = name
	}

	var items []domain.OrderItem
	var unknown []string

	for _, line := range parsed {
		if canonical, ok := catalog[strings.ToLower(line.Name)]; ok {
			items = append(items, domain.OrderItem{Product: canonical, Quantity: line.Quantity})
		} else {
			unknown = append(unknown, line.Name)
		}
	}

	if len(items) == 0 {
		return c.Send("Ни один продукт не найден в каталоге.\n\nПроверьте правильность названий.")
	}

	b.updateSession(c.Sender().ID, func(s *session) {
		s.location = location
		s.items = items
	})

	var sb strings.Builder
	sb.WriteString("<b>Предпросмотр batch-заявки</b>")
	if location != "" {
		sb.WriteString(fmt.Sprintf("\n%s", html.EscapeString(location)))
	}
	sb.WriteString(fmt.Sprintf("\n\n<b>Распознано позиций: %d</b>\n", len(items)))
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("• %s — %d шт.\n", html.EscapeString(it.Product), it.Quantity))
	}
	if len(unknown) > 0 {
		sb.WriteString(fmt.Sprintf("\n<b>Ошибки: не найдены в каталоге (%d)</b>\n", len(unknown)))
		for _, name := range unknown {
			sb.WriteString(fmt.Sprintf("— %s\n", html.EscapeString(name)))
		}
		sb.WriteString("\nБудут отправлены только распознанные позиции. Если нужно исправить, отправьте новый заказ целиком.")
	} else {
		sb.WriteString("\nОшибок нет.")
	}
	sb.WriteString("\n\n<blockquote expandable>")
	sb.WriteString("<b>Текст для копирования</b>\n")
	sb.WriteString(html.EscapeString(formatCopyableOrder(location, items, unknown)))
	sb.WriteString("</blockquote>")

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("📤 Отправить", "confirm"),
			markup.Data("❌ Отмена", "cancel_cb"),
		),
	)
	return c.Send(sb.String(), tele.ModeHTML, markup)
}

func formatCopyableOrder(location string, items []domain.OrderItem, unknown []string) string {
	var sb strings.Builder
	if location != "" {
		sb.WriteString(location)
		sb.WriteString("\n")
	}
	if location != "" {
		sb.WriteString("\n")
	}
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("%s %d\n", it.Product, it.Quantity))
	}
	if len(unknown) > 0 {
		sb.WriteString("\nНЕ НАЙДЕНЫ В КАТАЛОГЕ:\n")
		for _, name := range unknown {
			sb.WriteString(name)
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

func formatMultiOrderSummary(orders []domain.Order, notFound []string, dough []domain.DoughSummary) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📦 *Сводка по %d заказам*\n\n", len(orders)))

	for _, o := range orders {
		sb.WriteString(fmt.Sprintf("`%s` (%s)\n", o.Number, o.CreatedAt.Local().Format("02.01 15:04")))
		for _, it := range o.Items {
			sb.WriteString(fmt.Sprintf("  • %s × %d\n", it.Product, it.Quantity))
		}
	}

	if len(notFound) > 0 {
		sb.WriteString("\n⚠️ *Не найдены:*\n")
		for _, n := range notFound {
			sb.WriteString(fmt.Sprintf("  — `%s`\n", n))
		}
	}

	if len(dough) > 0 {
		sort.Slice(dough, func(i, j int) bool { return dough[i].DoughName < dough[j].DoughName })
		sb.WriteString("\n*Итого тесто:*\n")
		for _, d := range dough {
			sb.WriteString(fmt.Sprintf("🧁 %s: *%.3f кг*\n", d.DoughName, d.TotalKg))
		}
	} else {
		sb.WriteString("\nТесто по заявкам не требуется.")
	}
	return sb.String()
}

func formatOrderSummary(order domain.Order, dough []domain.DoughSummary) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ *Заказ %s*\n", order.Number))
	sb.WriteString(fmt.Sprintf("🕐 %s\n\n", order.CreatedAt.Local().Format("02.01.2006 15:04")))
	sb.WriteString("*Состав заявки:*\n")
	for _, it := range order.Items {
		sb.WriteString(fmt.Sprintf("• %s — %d шт.\n", it.Product, it.Quantity))
	}
	if len(dough) > 0 {
		sort.Slice(dough, func(i, j int) bool { return dough[i].DoughName < dough[j].DoughName })
		sb.WriteString("\n*Потребность в тесте:*\n")
		for _, d := range dough {
			sb.WriteString(fmt.Sprintf("🧁 %s: *%.3f кг*\n", d.DoughName, d.TotalKg))
		}
	} else {
		sb.WriteString("\nТесто по заявке не требуется.")
	}
	return sb.String()
}
