package booking

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jung-kurt/gofpdf"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func InvoicePDFHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoiceID, ok := parseInvoiceID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		invoice, err := deps.Store.GetInvoiceForPDF(ctx, invoiceID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if user.Role != "admin" && invoice.TenantID != user.UserID && invoice.OwnerID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}
		if invoice.Status == "void" {
			response.Fail(w, http.StatusBadRequest, "Tagihan sudah dibatalkan.", nil)
			return
		}

		var orderID, paymentType string
		if invoice.Status == "paid" {
			payment, err := deps.Store.GetSettledPaymentForInvoice(ctx, invoice.ID)
			switch {
			case errors.Is(err, pgx.ErrNoRows):
			case err != nil:
				response.InternalErrorHandler(w, r)
				return
			default:
				orderID = payment.OrderID
				paymentType = payment.PaymentType.String
			}
		}

		pdf := renderInvoicePDF(invoice, orderID, paymentType)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, invoiceFilename(invoice.Number)))
		if err := pdf.Output(w); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
	}
}

func invoiceFilename(number string) string {
	out := make([]byte, 0, len(number))
	for _, c := range number {
		if c == '/' {
			out = append(out, '-')
			continue
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func maskPhone(phone string) string {
	if len(phone) <= 8 {
		return phone
	}
	return phone[:4] + "****" + phone[len(phone)-4:]
}

func renderInvoicePDF(invoice dbgen.GetInvoiceForPDFRow, orderID, paymentType string) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "Sewa Kos", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, "Nomor: "+invoice.Number, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Tanggal terbit: "+money.FormatDateWIB(invoice.IssuedAt.Time), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(0, 6, "Penyewa", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, invoice.TenantName+" ("+maskPhone(invoice.TenantPhone)+")", "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, invoice.PropertyName+" - "+invoice.RoomName, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Periode: %s - %s", money.FormatDateWIB(invoice.PeriodStart.Time), money.FormatDateWIB(invoice.PeriodEnd.Time)), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	addInvoiceLine(pdf, "Sewa", invoice.RentAmount)
	if invoice.Discount > 0 {
		addInvoiceLine(pdf, "Diskon", -invoice.Discount)
	}
	if invoice.Deposit > 0 {
		addInvoiceLine(pdf, "Deposit", invoice.Deposit)
	}
	addInvoiceLine(pdf, "Biaya admin", invoice.AdminFee)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(140, 7, "Total", "T", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, money.FormatRupiah(invoice.Total), "T", 1, "R", false, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	if invoice.Status == "paid" {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(0, 128, 0)
		pdf.CellFormat(0, 8, "LUNAS", "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, "Tanggal bayar: "+money.FormatHumanWIB(invoice.PaidAt.Time), "", 1, "L", false, 0, "")
		if paymentType != "" {
			pdf.CellFormat(0, 6, "Metode: "+paymentType, "", 1, "L", false, 0, "")
		}
		if orderID != "" {
			pdf.CellFormat(0, 6, "Order ID: "+orderID, "", 1, "L", false, 0, "")
		}
	} else {
		pdf.CellFormat(0, 6, "Jatuh tempo: "+money.FormatDateWIB(invoice.DueDate.Time), "", 1, "L", false, 0, "")
	}

	return pdf
}

func addInvoiceLine(pdf *gofpdf.Fpdf, label string, amount int64) {
	pdf.CellFormat(140, 7, label, "", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, money.FormatRupiah(amount), "", 1, "R", false, 0, "")
}
