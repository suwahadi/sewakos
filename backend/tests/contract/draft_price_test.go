package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/money"
)

type draftPriceItem struct {
	Type                  string `json:"type"`
	Price                 int64  `json:"price"`
	Deposit               int64  `json:"deposit"`
	AdminFee              int64  `json:"admin_fee"`
	FlashSaleDiscount     int64  `json:"flash_sale_discount"`
	TotalOffFirstPayment  int64  `json:"total_off_first_payment"`
	RemainingPayment      int64  `json:"remaining_payment"`
	RemainingPaymentLabel string `json:"remaining_payment_label"`
}

func fetchDraftPrice(t *testing.T, srv *httptest.Server, path string) map[string]draftPriceItem {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, path, "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("draft price %s: status = %d, body = %s", path, resp.StatusCode, body)
	}
	var parsed struct {
		Data []draftPriceItem `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse draft price: %v", err)
	}
	items := map[string]draftPriceItem{}
	for _, item := range parsed.Data {
		items[item.Type] = item
	}
	return items
}

func TestDraftPriceIsPublicAndAppliesFlashSaleAndDuration(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ctx := context.Background()
	roomID := seedBookingFixture(t, pool, 2250000, 2, true)
	if _, err := pool.Exec(ctx, `UPDATE rooms SET price_daily = 100000 WHERE id = $1`, roomID); err != nil {
		t.Fatal(err)
	}
	var adminFee int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE((SELECT (value #>> '{}')::bigint FROM settings WHERE key = 'admin_fee'), 0)`).Scan(&adminFee); err != nil {
		t.Fatal(err)
	}

	items := fetchDraftPrice(t, srv, fmt.Sprintf("/api/booking/draft-price/%d?is_flash_sale=true&duration=3", roomID))
	if _, ok := items["weekly"]; ok || len(items) != 2 {
		t.Fatalf("items = %+v, want monthly and daily only (no weekly price)", items)
	}
	monthly := items["monthly"]
	if monthly.Price != 2250000 || monthly.Deposit != 2250000 || monthly.AdminFee != adminFee || monthly.FlashSaleDiscount != 250000 ||
		monthly.TotalOffFirstPayment != 2250000*2+adminFee-250000 || monthly.RemainingPayment != 2250000*2+adminFee {
		t.Errorf("monthly = %+v, want one period regardless of duration", monthly)
	}
	daily := items["daily"]
	if daily.TotalOffFirstPayment != 100000*4+adminFee-250000 || daily.RemainingPayment != 100000*4+adminFee ||
		daily.RemainingPaymentLabel != money.FormatRupiah(100000*4+adminFee) {
		t.Errorf("daily = %+v, want three days plus one day deposit paid up front", daily)
	}

	items = fetchDraftPrice(t, srv, fmt.Sprintf("/api/booking/draft-price/%d", roomID))
	if items["monthly"].FlashSaleDiscount != 0 || items["monthly"].TotalOffFirstPayment != 2250000*2+adminFee {
		t.Errorf("without is_flash_sale = %+v, want no discount", items["monthly"])
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/booking/draft-price/999999999", "", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown room: status = %d, body = %s, want 404", resp.StatusCode, body)
	}
}
