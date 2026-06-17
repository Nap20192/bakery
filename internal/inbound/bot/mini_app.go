package bot

import (
	"net/url"
	"strings"

	tele "gopkg.in/telebot.v3"
)

const (
	miniAppModeOrders  = "orders"
	miniAppModeCreate  = "create"
	miniAppModeView    = "view"
	miniAppModeEdit    = "edit"
	miniAppModeMonitor = "monitor"
)

func (b *OrderBot) miniAppLink(mode, orderNumber string, orderNumbers []string) string {
	if b == nil || b.baseBot == nil || strings.TrimSpace(b.miniAppURL) == "" {
		return ""
	}
	link, err := url.Parse(b.miniAppURL)
	if err != nil {
		return ""
	}
	query := link.Query()
	query.Del("mode")
	query.Del("order")
	query.Del("orders")
	if mode = strings.TrimSpace(mode); mode != "" {
		query.Set("mode", mode)
	}
	if orderNumber = strings.TrimSpace(orderNumber); orderNumber != "" {
		query.Set("order", orderNumber)
	}
	for _, number := range orderNumbers {
		if number = strings.TrimSpace(number); number != "" {
			query.Add("orders", number)
		}
	}
	link.RawQuery = query.Encode()
	return link.String()
}

func (b *OrderBot) miniAppButton(markup *tele.ReplyMarkup, label, mode, orderNumber string, orderNumbers []string) (tele.Btn, bool) {
	link := b.miniAppLink(mode, orderNumber, orderNumbers)
	if link == "" {
		return tele.Btn{}, false
	}
	return markup.WebApp(label, &tele.WebApp{URL: link}), true
}
