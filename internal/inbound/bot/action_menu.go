package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	accessdomain "bakery/internal/domain/access"
	"bakery/internal/pkg/enum"

	tele "gopkg.in/telebot.v3"
)

const (
	orderFilterTodayText     = "Сегодня"
	orderFilterTomorrowText  = "Завтра"
	orderFilterAllDatesText  = "Все даты"
	orderFilterAllShopsText  = "Все магазины"
	orderReplyShopButtonSize = 2
)

const (
	actionTemplates   = "Шаблоны"
	actionOrders      = "Последние заказы"
	actionSubmitOrder = "Отправить заказ"
	actionUpdateOrder = "Обновить заказ"
	actionCancelOrder = "Отменить заказ"
	actionSync        = "Sync iiko"
)

type actionMenuState string

const (
	actionStateGuest             actionMenuState = "guest"
	actionStateShopIdle          actionMenuState = "shop_idle"
	actionStateShopCreate        actionMenuState = "shop_create"
	actionStateShopUpdate        actionMenuState = "shop_update"
	actionStateWorkshopIdle      actionMenuState = "workshop_idle"
	actionStateWorkshopFilter    actionMenuState = "workshop_filter"
	actionStateAdminIdle         actionMenuState = "admin_idle"
	actionStateAdminCreate       actionMenuState = "admin_create"
	actionStateAdminUpdate       actionMenuState = "admin_update"
	actionStateAdminWorkshop     actionMenuState = "admin_workshop"
	actionStateAdminWorkshopFilt actionMenuState = "admin_workshop_filter"
)

type actionMenuSnapshot struct {
	state          actionMenuState
	admin          bool
	departmentType string
	orderItems     int
	editOrder      string
	filterShop     string
	filterDate     time.Time
}

func (b *OrderBot) actionMarkup(c tele.Context) *tele.ReplyMarkup {
	return b.actionKeyboard(b.actionMenuRows(c)...)
}

func (b *OrderBot) actionMenu(c tele.Context) actionMenuSnapshot {
	user, ok := b.currentUser(c)
	if !ok {
		return actionMenuSnapshot{state: actionStateGuest}
	}

	snapshot := actionMenuSnapshot{
		admin:          strings.EqualFold(user.Role, accessdomain.RoleAdmin),
		departmentType: b.userDepartmentType(c, user),
	}
	var filterShopID *int64
	if c.Sender() != nil {
		b.mu.Lock()
		if s := b.sessions[c.Sender().ID]; s != nil {
			snapshot.orderItems = len(s.items)
			snapshot.editOrder = strings.TrimSpace(s.editOrderNumber)
			snapshot.filterDate = s.orderFilter.FulfillmentDate
			filterShopID = cloneInt64Ptr(s.orderFilter.FromDepartmentID)
		}
		b.mu.Unlock()
	}
	if filterShopID != nil && b.departmentSvc != nil {
		snapshot.filterShop = b.departmentDisplayName(requestContext(c), filterShopID)
	}
	if strings.TrimSpace(snapshot.filterShop) == "" {
		snapshot.filterShop = orderFilterAllShopsText
	}

	snapshot.state = snapshot.resolveState()
	return snapshot
}

func (s actionMenuSnapshot) resolveState() actionMenuState {
	if s.admin {
		if s.departmentType == string(enum.DepartmentTypeWorkshop) {
			if s.hasFilter() {
				return actionStateAdminWorkshopFilt
			}
			return actionStateAdminWorkshop
		}
		if s.editOrder != "" && s.orderItems > 0 {
			return actionStateAdminUpdate
		}
		if s.orderItems > 0 {
			return actionStateAdminCreate
		}
		return actionStateAdminIdle
	}

	if s.departmentType == string(enum.DepartmentTypeWorkshop) {
		if s.hasFilter() {
			return actionStateWorkshopFilter
		}
		return actionStateWorkshopIdle
	}
	if s.editOrder != "" && s.orderItems > 0 {
		return actionStateShopUpdate
	}
	if s.orderItems > 0 {
		return actionStateShopCreate
	}
	return actionStateShopIdle
}

func (s actionMenuSnapshot) hasFilter() bool {
	return !s.filterDate.IsZero() || !strings.EqualFold(strings.TrimSpace(s.filterShop), orderFilterAllShopsText)
}

func (s actionMenuSnapshot) rows() [][]string {
	switch s.state {
	case actionStateGuest:
		return nil
	case actionStateWorkshopIdle, actionStateWorkshopFilter:
		return append(workshopActionRows(), s.filterRows()...)
	case actionStateAdminWorkshop, actionStateAdminWorkshopFilt:
		rows := append(workshopActionRows(), []string{actionSync})
		return append(rows, s.filterRows()...)
	case actionStateAdminIdle:
		return append(shopActionRows(), []string{actionSync})
	case actionStateAdminCreate, actionStateAdminUpdate:
		return append(append(shopActionRows(), []string{actionSync}), s.orderRows()...)
	case actionStateShopCreate, actionStateShopUpdate:
		return append(shopActionRows(), s.orderRows()...)
	case actionStateShopIdle:
		fallthrough
	default:
		return shopActionRows()
	}
}

func shopActionRows() [][]string {
	return [][]string{{actionTemplates, actionOrders}}
}

func workshopActionRows() [][]string {
	return [][]string{{actionOrders}}
}

func (s actionMenuSnapshot) orderRows() [][]string {
	if s.orderItems <= 0 {
		return nil
	}
	stateText, action := s.orderStateText()
	return [][]string{{stateText}, {action}, {actionCancelOrder}}
}

func (s actionMenuSnapshot) orderStateText() (string, string) {
	if strings.TrimSpace(s.editOrder) != "" {
		return fmt.Sprintf("Редактируется: %s", s.editOrder), actionUpdateOrder
	}
	return fmt.Sprintf("Создается заказ: %d поз.", s.orderItems), actionSubmitOrder
}

func (s actionMenuSnapshot) filterRows() [][]string {
	return [][]string{
		{s.filterStateText()},
		{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText},
	}
}

func (s actionMenuSnapshot) filterStateText() string {
	date := orderFilterAllDatesText
	if !s.filterDate.IsZero() {
		date = s.filterDate.Format("02.01.2006")
	}
	return fmt.Sprintf("Фильтр: %s / %s", s.filterShop, date)
}

func (b *OrderBot) actionMenuRows(c tele.Context) [][]string {
	snapshot := b.actionMenu(c)
	rows := snapshot.rows()
	if snapshot.departmentType != string(enum.DepartmentTypeWorkshop) {
		return rows
	}
	return append(rows, b.orderShopFilterReplyRows(requestContext(c))...)
}

func (b *OrderBot) isCurrentOrderStateButton(c tele.Context, text string) bool {
	snapshot := b.actionMenu(c)
	stateText, _ := snapshot.orderStateText()
	return snapshot.orderItems > 0 && strings.EqualFold(strings.TrimSpace(text), stateText)
}

func (b *OrderBot) isOrderFilterStateButton(c tele.Context, text string) bool {
	snapshot := b.actionMenu(c)
	if snapshot.departmentType != string(enum.DepartmentTypeWorkshop) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(text), snapshot.filterStateText())
}

func (b *OrderBot) orderShopFilterReplyRows(ctx context.Context) [][]string {
	if b.departmentSvc == nil {
		return nil
	}
	shops, err := b.departmentSvc.ListByType(ctx, enum.DepartmentTypeShop)
	if err != nil {
		slog.WarnContext(ctx, "list shop departments for order filters failed", "error", err)
		return nil
	}
	buttons := make([]string, 0, len(shops)+1)
	buttons = append(buttons, orderFilterAllShopsText)
	for _, shop := range shops {
		buttons = append(buttons, shop.Name)
	}
	rows := make([][]string, 0, (len(buttons)+1)/orderReplyShopButtonSize)
	for len(buttons) > 0 {
		end := orderReplyShopButtonSize
		if len(buttons) < end {
			end = len(buttons)
		}
		rows = append(rows, buttons[:end])
		buttons = buttons[end:]
	}
	return rows
}

func (b *OrderBot) actionKeyboard(rows ...[]string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	replyRows := make([]tele.Row, 0, len(rows))
	for _, labels := range rows {
		buttons := make([]tele.Btn, 0, len(labels))
		for _, label := range labels {
			switch label {
			case actionTemplates:
				if button, ok := b.miniAppButton(markup, "Новый заказ", miniAppModeCreate, "", nil); ok {
					buttons = append(buttons, button)
					continue
				}
			case actionOrders:
				if button, ok := b.miniAppButton(markup, label, miniAppModeOrders, "", nil); ok {
					buttons = append(buttons, button)
					continue
				}
			}
			buttons = append(buttons, markup.Text(label))
		}
		replyRows = append(replyRows, markup.Row(buttons...))
	}
	markup.Reply(replyRows...)
	return markup
}

func (b *OrderBot) currentUser(c tele.Context) (accessdomain.AuthUser, bool) {
	if raw := c.Get(authUserContextKey); raw != nil {
		if user, ok := raw.(accessdomain.AuthUser); ok {
			return user, true
		}
	}
	if b.authSvc == nil || c.Sender() == nil {
		return accessdomain.AuthUser{}, false
	}
	user, err := b.authSvc.GetUserByTelegramID(requestContext(c), c.Sender().ID)
	if err != nil {
		return accessdomain.AuthUser{}, false
	}
	c.Set(authUserContextKey, user)
	return user, true
}

func (b *OrderBot) userDepartmentType(c tele.Context, user accessdomain.AuthUser) string {
	if b.departmentSvc == nil || user.DepartmentID == nil {
		return ""
	}
	department, err := b.departmentSvc.GetByID(requestContext(c), *user.DepartmentID)
	if err != nil {
		return ""
	}
	return department.Type
}
