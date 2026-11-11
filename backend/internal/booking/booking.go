package booking

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type Store interface {
	GetRoomForDraftPrice(ctx context.Context, id int64) (dbgen.GetRoomForDraftPriceRow, error)
	GetRoomForSubmit(ctx context.Context, id int64) (dbgen.GetRoomForSubmitRow, error)
	ListBookingsForUser(ctx context.Context, arg dbgen.ListBookingsForUserParams) ([]dbgen.ListBookingsForUserRow, error)
	GetBookingByID(ctx context.Context, arg dbgen.GetBookingByIDParams) (dbgen.GetBookingByIDRow, error)
	GetInitialInvoiceForBooking(ctx context.Context, bookingID int64) (dbgen.GetInitialInvoiceForBookingRow, error)
	GetLatestPaymentForInvoice(ctx context.Context, invoiceID int64) (dbgen.GetLatestPaymentForInvoiceRow, error)
	GetRefundForInvoice(ctx context.Context, invoiceID int64) (dbgen.GetRefundForInvoiceRow, error)
}

type AdminFee interface {
	Get(ctx context.Context) (int64, error)
}

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Deps struct {
	Store                Store
	DB                   TxBeginner
	AdminFee             AdminFee
	Midtrans             midtrans.Client
	PaymentExpiryMinutes int
	MidtransFinishURL    string
}
