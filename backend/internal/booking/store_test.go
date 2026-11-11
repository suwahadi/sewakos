package booking

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type baseStubStore struct{}

func (baseStubStore) GetRoomForDraftPrice(ctx context.Context, id int64) (dbgen.GetRoomForDraftPriceRow, error) {
	return dbgen.GetRoomForDraftPriceRow{}, pgx.ErrNoRows
}

func (baseStubStore) GetRoomForSubmit(ctx context.Context, id int64) (dbgen.GetRoomForSubmitRow, error) {
	return dbgen.GetRoomForSubmitRow{}, pgx.ErrNoRows
}

func (baseStubStore) ListBookingsForUser(ctx context.Context, arg dbgen.ListBookingsForUserParams) ([]dbgen.ListBookingsForUserRow, error) {
	return nil, nil
}

func (baseStubStore) GetBookingByID(ctx context.Context, arg dbgen.GetBookingByIDParams) (dbgen.GetBookingByIDRow, error) {
	return dbgen.GetBookingByIDRow{}, pgx.ErrNoRows
}

func (baseStubStore) GetInitialInvoiceForBooking(ctx context.Context, bookingID int64) (dbgen.GetInitialInvoiceForBookingRow, error) {
	return dbgen.GetInitialInvoiceForBookingRow{}, pgx.ErrNoRows
}

func (baseStubStore) GetLatestPaymentForInvoice(ctx context.Context, invoiceID int64) (dbgen.GetLatestPaymentForInvoiceRow, error) {
	return dbgen.GetLatestPaymentForInvoiceRow{}, pgx.ErrNoRows
}

func (baseStubStore) GetRefundForInvoice(ctx context.Context, invoiceID int64) (dbgen.GetRefundForInvoiceRow, error) {
	return dbgen.GetRefundForInvoiceRow{}, pgx.ErrNoRows
}

func (baseStubStore) ListInvoicesForTenant(ctx context.Context, arg dbgen.ListInvoicesForTenantParams) ([]dbgen.ListInvoicesForTenantRow, error) {
	return nil, nil
}

func (baseStubStore) GetInvoiceForTenant(ctx context.Context, id int64) (dbgen.GetInvoiceForTenantRow, error) {
	return dbgen.GetInvoiceForTenantRow{}, pgx.ErrNoRows
}

func (baseStubStore) GetInvoiceForPDF(ctx context.Context, id int64) (dbgen.GetInvoiceForPDFRow, error) {
	return dbgen.GetInvoiceForPDFRow{}, pgx.ErrNoRows
}

func (baseStubStore) GetSettledPaymentForInvoice(ctx context.Context, invoiceID int64) (dbgen.GetSettledPaymentForInvoiceRow, error) {
	return dbgen.GetSettledPaymentForInvoiceRow{}, pgx.ErrNoRows
}
