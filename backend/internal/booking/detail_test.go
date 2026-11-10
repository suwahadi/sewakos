package booking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type stubDetailStore struct {
	baseStubStore
	booking dbgen.GetBookingByIDRow
	err     error
	invoice dbgen.GetInitialInvoiceForBookingRow
}

func (s stubDetailStore) GetBookingByID(ctx context.Context, arg dbgen.GetBookingByIDParams) (dbgen.GetBookingByIDRow, error) {
	return s.booking, s.err
}

func (s stubDetailStore) GetInitialInvoiceForBooking(ctx context.Context, bookingID int64) (dbgen.GetInitialInvoiceForBookingRow, error) {
	return s.invoice, nil
}

func detailRequest(t *testing.T, deps Deps, user authctx.User, bookingID string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/booking/{booking}", DetailHandler(deps, user))
	req := httptest.NewRequest(http.MethodGet, "/booking/"+bookingID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDetailHandlerNotFound(t *testing.T) {
	deps := Deps{Store: stubDetailStore{err: pgx.ErrNoRows}}
	w := detailRequest(t, deps, authctx.User{UserID: 1}, "99")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestDetailHandlerForbiddenForOtherTenant(t *testing.T) {
	deps := Deps{Store: stubDetailStore{booking: dbgen.GetBookingByIDRow{ID: 1, TenantID: 2}}}
	w := detailRequest(t, deps, authctx.User{UserID: 1}, "1")
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}
