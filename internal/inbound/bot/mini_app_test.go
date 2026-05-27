package bot

import (
	"net/url"
	"reflect"
	"testing"
)

func TestMiniAppLinkIncludesRequestedScreenAndOrders(t *testing.T) {
	b := &OrderBot{baseBot: &baseBot{miniAppURL: "https://orders.example/app?source=bot&mode=old"}}

	got := b.miniAppLink(miniAppModeMonitor, "", []string{"Г.27.05.26.001", "С.27.05.26.002"})
	link, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse mini app link: %v", err)
	}
	query := link.Query()
	if query.Get("source") != "bot" || query.Get("mode") != miniAppModeMonitor {
		t.Fatalf("query = %v", query)
	}
	wantOrders := []string{"Г.27.05.26.001", "С.27.05.26.002"}
	if gotOrders := query["orders"]; !reflect.DeepEqual(gotOrders, wantOrders) {
		t.Fatalf("orders = %v, want %v", gotOrders, wantOrders)
	}
	if query.Get("order") != "" {
		t.Fatalf("order = %q, want empty", query.Get("order"))
	}
}

func TestMiniAppLinkReturnsEmptyWithoutConfiguredURL(t *testing.T) {
	b := &OrderBot{baseBot: &baseBot{}}
	if got := b.miniAppLink(miniAppModeCreate, "", nil); got != "" {
		t.Fatalf("miniAppLink = %q, want empty", got)
	}
}
