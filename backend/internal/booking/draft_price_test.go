package booking

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type stubDraftPriceStore struct {
	baseStubStore
	room dbgen.GetRoomForDraftPriceRow
	err  error
}

func (s stubDraftPriceStore) GetRoomForDraftPrice(ctx context.Context, id int64) (dbgen.GetRoomForDraftPriceRow, error) {
	return s.room, s.err
}

type stubAdminFee int64

func (s stubAdminFee) Get(ctx context.Context) (int64, error) { return int64(s), nil }

func draftPriceRequest(t *testing.T, deps Deps, roomID, query string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/booking/draft-price/{room}", DraftPriceHandler(deps))
	req := httptest.NewRequest(http.MethodGet, "/booking/draft-price/"+roomID+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDraftPriceHandlerMatchesDocumentedExample(t *testing.T) {
	deps := Deps{
		Store: stubDraftPriceStore{room: dbgen.GetRoomForDraftPriceRow{
			Name: "Kamar Tipe A", PriceMonthly: 2250000,
			IsFlashSale: true, FlashSaleDiscount: 250000,
		}},
		AdminFee: stubAdminFee(0),
	}
	w := draftPriceRequest(t, deps, "1", "?is_flash_sale=true")

	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1 (only monthly priced)", len(body.Data))
	}
	item := body.Data[0]
	if item["total_off_first_payment"] != float64(4250000) {
		t.Errorf("total_off_first_payment = %v, want 4250000", item["total_off_first_payment"])
	}
	if item["remaining_payment"] != float64(4500000) {
		t.Errorf("remaining_payment = %v, want 4500000", item["remaining_payment"])
	}
	if item["total_off_first_payment_label"] != "Rp4.250.000" {
		t.Errorf("total_off_first_payment_label = %v", item["total_off_first_payment_label"])
	}
}

func TestDraftPriceHandlerWithoutFlashSaleFlagAppliesNoDiscount(t *testing.T) {
	deps := Deps{
		Store: stubDraftPriceStore{room: dbgen.GetRoomForDraftPriceRow{
			PriceMonthly: 2250000, IsFlashSale: true, FlashSaleDiscount: 250000,
		}},
		AdminFee: stubAdminFee(0),
	}
	w := draftPriceRequest(t, deps, "1", "")

	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.Data[0]["flash_sale_discount"] != float64(0) {
		t.Errorf("flash_sale_discount = %v, want 0 when is_flash_sale query is absent", body.Data[0]["flash_sale_discount"])
	}
}

func TestDraftPriceHandlerDurationAffectsDailyAndWeeklyOnly(t *testing.T) {
	deps := Deps{
		Store: stubDraftPriceStore{room: dbgen.GetRoomForDraftPriceRow{
			PriceMonthly: 2250000,
			PriceWeekly:  pgtype.Int8{Int64: 500000, Valid: true},
			PriceDaily:   pgtype.Int8{Int64: 100000, Valid: true},
		}},
		AdminFee: stubAdminFee(0),
	}
	w := draftPriceRequest(t, deps, "1", "?duration=3")

	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	byType := map[string]map[string]any{}
	for _, item := range body.Data {
		byType[item["type"].(string)] = item
	}

	if byType["monthly"]["remaining_payment"] != float64(2250000+2250000) {
		t.Errorf("monthly must ignore duration, got %v", byType["monthly"]["remaining_payment"])
	}
	if byType["weekly"]["remaining_payment"] != float64(500000*3+500000) {
		t.Errorf("weekly remaining_payment = %v, want %d", byType["weekly"]["remaining_payment"], 500000*3+500000)
	}
	if byType["daily"]["remaining_payment"] != float64(100000*3+100000) {
		t.Errorf("daily remaining_payment = %v, want %d", byType["daily"]["remaining_payment"], 100000*3+100000)
	}
}

func TestDraftPriceHandlerDurationOutOfRangeIsClamped(t *testing.T) {
	deps := Deps{
		Store: stubDraftPriceStore{room: dbgen.GetRoomForDraftPriceRow{
			PriceDaily: pgtype.Int8{Int64: 100000, Valid: true},
		}},
		AdminFee: stubAdminFee(0),
	}
	w := draftPriceRequest(t, deps, "1", "?duration=999")

	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	var daily map[string]any
	for _, item := range body.Data {
		if item["type"] == "daily" {
			daily = item
		}
	}
	if daily == nil {
		t.Fatalf("no daily item in data: %v", body.Data)
	}
	if daily["remaining_payment"] != float64(100000*30+100000) {
		t.Errorf("remaining_payment = %v, want duration clamped to 30", daily["remaining_payment"])
	}
}

func TestDraftPriceHandlerRoomNotFound(t *testing.T) {
	deps := Deps{Store: stubDraftPriceStore{err: pgx.ErrNoRows}, AdminFee: stubAdminFee(0)}
	w := draftPriceRequest(t, deps, "999", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
