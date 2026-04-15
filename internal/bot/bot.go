package bot

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"bakery/internal/app"
	"bakery/internal/domain"

	tele "gopkg.in/telebot.v3"
)

const (
	stateIdle     = "idle"
	stateAdding   = "adding"
	stateQuantity = "quantity"
	stateSearch   = "search"
	pageSize      = 14

	sessionTTL     = 12 * time.Hour // незавершённая заявка живёт 12 часов
	cleanupInterval = time.Hour     // проверяем каждый час
)

type session struct {
	state     string
	items     []domain.OrderItem
	selected  string
	page      int
	filter    string
	updatedAt time.Time // время последнего изменения
}

type Bot struct {
	tele        *tele.Bot
	orderSvc    *app.OrderService
	orderRepo   domain.OrderRepository
	productRepo domain.ProductRepository
	doughRepo   domain.DoughRepository

	mu           sync.Mutex
	sessions     map[int64]*session
	productNames []string
}

func New(
	token string,
	orderSvc *app.OrderService,
	orderRepo domain.OrderRepository,
	productRepo domain.ProductRepository,
	doughRepo domain.DoughRepository,
) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("bot: new: %w", err)
	}

	names, err := productRepo.List()
	if err != nil {
		return nil, fmt.Errorf("bot: load products: %w", err)
	}
	sort.Strings(names)

	bot := &Bot{
		tele:         b,
		orderSvc:     orderSvc,
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		doughRepo:    doughRepo,
		sessions:     make(map[int64]*session),
		productNames: names,
	}
	bot.register()
	return bot, nil
}

func (b *Bot) Start() {
	go b.cleanupLoop()
	b.tele.Start()
}

// updateSession атомарно изменяет сессию пользователя под мьютексом.
func (b *Bot) updateSession(uid int64, fn func(*session)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.sessions[uid]
	if !ok {
		s = &session{state: stateIdle}
		b.sessions[uid] = s
	}
	fn(s)
	s.updatedAt = time.Now()
}

// cleanupLoop удаляет сессии, которые не обновлялись дольше sessionTTL.
func (b *Bot) cleanupLoop() {
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
			log.Printf("SESSION cleanup: удалено %d истёкших сессий", len(expired))
		}
	}
}

// readSession возвращает копию сессии (чтение под мьютексом).
func (b *Bot) readSession(uid int64) session {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.sessions[uid]
	if !ok {
		return session{state: stateIdle}
	}
	// возвращаем копию, чтобы не держать мьютекс снаружи
	cp := *s
	cp.items = make([]domain.OrderItem, len(s.items))
	copy(cp.items, s.items)
	return cp
}

func (b *Bot) register() {
	bt := b.tele
	bt.Use(b.logMiddleware)

	bt.Handle("/start", b.handleStart)
	bt.Handle("/new", b.handleNew)
	bt.Handle("/orders", b.handleOrders)
	bt.Handle("/cancel", b.handleCancel)
	bt.Handle("/search", b.handleSearchCmd)

	bt.Handle("\fprod", b.handleProductCallback)
	bt.Handle("\fpage", b.handlePageCallback)
	bt.Handle("\fconfirm", b.handleConfirm)
	bt.Handle("\fcancel_cb", b.handleCancelCallback)
	bt.Handle("\fadd_more", b.handleAddMore)
	bt.Handle("\fsearch_cb", b.handleSearchCallback)

	bt.Handle(tele.OnText, b.handleText)
}

func (b *Bot) logMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		start := time.Now()
		sender := c.Sender()

		var kind, payload string
		if cb := c.Callback(); cb != nil {
			kind = "callback"
			payload = cb.Unique + " data=" + cb.Data
		} else if c.Message() != nil {
			kind = "message"
			payload = c.Text()
			if len(payload) > 80 {
				payload = payload[:80] + "…"
			}
		} else {
			kind = "update"
		}

		err := next(c)

		status := "ok"
		if err != nil {
			status = "err: " + err.Error()
		}
		log.Printf("[%s] uid=%d @%s %q → %s (%s)",
			kind, sender.ID, sender.Username, payload, status,
			time.Since(start).Round(time.Millisecond),
		)
		return err
	}
}

// --- handlers ---

func (b *Bot) handleStart(c tele.Context) error {
	return c.Send(
		"🍞 *Пекарня — расчёт теста*\n\n"+
			"/new — создать новую заявку\n"+
			"/search — поиск продукта по названию\n"+
			"/orders — последние заказы\n"+
			"/cancel — отменить текущую заявку",
		tele.ModeMarkdown,
	)
}

func (b *Bot) handleCancel(c tele.Context) error {
	b.updateSession(c.Sender().ID, func(s *session) {
		s.state = stateIdle
		s.items = nil
		s.selected = ""
		s.filter = ""
	})
	return c.Send("Заявка отменена.")
}

func (b *Bot) handleNew(c tele.Context) error {
	b.updateSession(c.Sender().ID, func(s *session) {
		s.state = stateAdding
		s.items = nil
		s.selected = ""
		s.page = 0
		s.filter = ""
	})
	snap := b.readSession(c.Sender().ID)
	// /new — всегда шлём новое сообщение (не редактируем)
	return b.renderPage(c, snap, "Выберите продукцию:", false)
}

func (b *Bot) handleSearchCmd(c tele.Context) error {
	b.updateSession(c.Sender().ID, func(s *session) {
		if s.state == stateIdle {
			s.items = nil
		}
		s.state = stateSearch
	})
	return c.Send("Введите часть названия продукта для поиска:")
}

func (b *Bot) handleSearchCallback(c tele.Context) error {
	b.updateSession(c.Sender().ID, func(s *session) {
		s.state = stateSearch
	})
	_ = c.Respond()
	// редактируем текущее сообщение, убирая клавиатуру
	_, _ = b.tele.Edit(c.Callback().Message, "Введите часть названия продукта для поиска:")
	return nil
}

func (b *Bot) handleAddMore(c tele.Context) error {
	b.updateSession(c.Sender().ID, func(s *session) {
		s.state = stateAdding
		s.page = 0
		s.filter = ""
	})
	snap := b.readSession(c.Sender().ID)
	_ = c.Respond()
	// редактируем существующее сообщение (не плодим новые)
	return b.renderPage(c, snap, "Добавьте ещё продукцию:", true)
}

// renderPage строит клавиатуру выбора продукта.
// edit=true → редактирует сообщение (из callback), edit=false → шлёт новое.
func (b *Bot) renderPage(c tele.Context, snap session, title string, edit bool) error {
	names := b.filtered(snap.filter)
	total := len(names)
	if total == 0 {
		return c.Send("Ничего не найдено. Попробуйте другой запрос.")
	}

	page := snap.page
	totalPages := (total + pageSize - 1) / pageSize
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	chunk := names[start:end]

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	for i := 0; i < len(chunk); i += 2 {
		idx1 := start + i
		btn1 := markup.Data(chunk[i], "prod", strconv.Itoa(idx1))
		if i+1 < len(chunk) {
			idx2 := start + i + 1
			btn2 := markup.Data(chunk[i+1], "prod", strconv.Itoa(idx2))
			rows = append(rows, markup.Row(btn1, btn2))
		} else {
			rows = append(rows, markup.Row(btn1))
		}
	}

	var navRow tele.Row
	if page > 0 {
		navRow = append(navRow, markup.Data("◀ Назад", "page", strconv.Itoa(page-1)))
	}
	navRow = append(navRow, markup.Data(fmt.Sprintf("%d/%d", page+1, totalPages), "page", strconv.Itoa(page)))
	if page < totalPages-1 {
		navRow = append(navRow, markup.Data("Вперёд ▶", "page", strconv.Itoa(page+1)))
	}
	rows = append(rows, navRow)
	rows = append(rows, markup.Row(markup.Data("🔍 Поиск", "search_cb")))

	if len(snap.items) > 0 {
		rows = append(rows, markup.Row(markup.Data("✅ Подтвердить заявку", "confirm")))
	}

	markup.Inline(rows...)

	header := title
	if snap.filter != "" {
		header += fmt.Sprintf(" (поиск: «%s»)", snap.filter)
	}

	if edit && c.Callback() != nil {
		// редактируем то же сообщение — старые кнопки заменяются, не накапливаются
		_, err := b.tele.Edit(c.Callback().Message, header, markup)
		return err
	}
	return c.Send(header, markup)
}

func (b *Bot) handlePageCallback(c tele.Context) error {
	page, err := strconv.Atoi(c.Data())
	if err != nil {
		return c.Respond()
	}
	b.updateSession(c.Sender().ID, func(s *session) {
		s.page = page
	})
	snap := b.readSession(c.Sender().ID)
	_ = c.Respond()
	// EDIT — не шлём новое сообщение, редактируем текущее
	return b.renderPage(c, snap, "Выберите продукцию:", true)
}

func (b *Bot) handleProductCallback(c tele.Context) error {
	idx, err := strconv.Atoi(c.Data())
	if err != nil || idx < 0 || idx >= len(b.productNames) {
		return c.Respond()
	}

	productName := b.productNames[idx]
	b.updateSession(c.Sender().ID, func(s *session) {
		s.state = stateQuantity
		s.selected = productName
	})

	_ = c.Respond()
	return c.Send(
		fmt.Sprintf("Продукт: *%s*\nВведите количество (штук):", productName),
		tele.ModeMarkdown,
	)
}

func (b *Bot) handleText(c tele.Context) error {
	// читаем состояние под мьютексом
	snap := b.readSession(c.Sender().ID)
	text := strings.TrimSpace(c.Text())

	switch snap.state {
	case stateSearch:
		b.updateSession(c.Sender().ID, func(s *session) {
			s.filter = text
			s.page = 0
			s.state = stateAdding
		})
		snap = b.readSession(c.Sender().ID)
		return b.renderPage(c, snap, "Результаты поиска:", false)

	case stateQuantity:
		qty, err := strconv.Atoi(text)
		if err != nil || qty <= 0 {
			return c.Send("Пожалуйста, введите целое положительное число.")
		}
		b.updateSession(c.Sender().ID, func(s *session) {
			s.items = append(s.items, domain.OrderItem{
				Product:  s.selected,
				Quantity: qty,
			})
			s.state = stateAdding
		})
		snap = b.readSession(c.Sender().ID)
		return b.sendCurrentOrder(c, snap)

	default:
		if len(splitNumbers(text)) > 0 {
			return b.handleOrderLookup(c, text)
		}
		if strings.HasPrefix(text, "/") {
			return c.Send("Неизвестная команда 🤷\n\n/start — показать меню")
		}
		return c.Send("Не понимаю сообщение 🤷\n\n" +
			"Отправьте номер заказа (например `15042026_ORDER_0001`)\n" +
			"или используйте /new чтобы создать заявку.",
			tele.ModeMarkdown,
		)
	}
}

func (b *Bot) handleOrderLookup(c tele.Context, text string) error {
	// разбиваем по пробелам/переносам — поддержка нескольких номеров
	numbers := splitNumbers(text)

	if len(numbers) == 1 {
		return b.showSingleOrder(c, numbers[0])
	}
	return b.showMultipleOrders(c, numbers)
}

func (b *Bot) showSingleOrder(c tele.Context, number string) error {
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

func (b *Bot) showMultipleOrders(c tele.Context, numbers []string) error {
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

	// объединяем все позиции для расчёта теста
	combined := b.orderSvc.CombineOrders(found)
	dough, err := b.orderSvc.CalculateDough(combined, b.doughRepo)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка расчёта теста: %v", err))
	}

	return c.Send(formatMultiOrderSummary(found, notFound, dough), tele.ModeMarkdown)
}

// splitNumbers разбивает строку на номера заказов (по пробелам/переносам).
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

func (b *Bot) sendCurrentOrder(c tele.Context, snap session) error {
	var sb strings.Builder
	sb.WriteString("📋 *Текущая заявка:*\n")
	for _, it := range snap.items {
		sb.WriteString(fmt.Sprintf("• %s — %d шт.\n", it.Product, it.Quantity))
	}

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("➕ Добавить ещё", "add_more"),
			markup.Data("✅ Подтвердить", "confirm"),
		),
		markup.Row(
			markup.Data("❌ Отменить", "cancel_cb"),
		),
	)
	return c.Send(sb.String(), tele.ModeMarkdown, markup)
}

func (b *Bot) handleConfirm(c tele.Context) error {
	var items []domain.OrderItem
	b.updateSession(c.Sender().ID, func(s *session) {
		items = make([]domain.OrderItem, len(s.items))
		copy(items, s.items)
		s.state = stateIdle
		s.items = nil
	})

	_ = c.Respond()
	if len(items) == 0 {
		return c.Send("Заявка пустая.")
	}

	// только здесь заказ записывается в SQLite
	order, err := b.orderRepo.Create(items)
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

func (b *Bot) handleCancelCallback(c tele.Context) error {
	b.updateSession(c.Sender().ID, func(s *session) {
		s.state = stateIdle
		s.items = nil
	})
	_ = c.Respond()
	return c.Send("Заявка отменена.")
}

func (b *Bot) handleOrders(c tele.Context) error {
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

func (b *Bot) filtered(filter string) []string {
	if filter == "" {
		return b.productNames
	}
	f := strings.ToLower(filter)
	var result []string
	for _, name := range b.productNames {
		if strings.Contains(strings.ToLower(name), f) {
			result = append(result, name)
		}
	}
	return result
}

func formatMultiOrderSummary(orders []domain.Order, notFound []string, dough []domain.DoughSummary) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📦 *Сводка по %d заказам*\n\n", len(orders)))

	// перечисляем заказы
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
