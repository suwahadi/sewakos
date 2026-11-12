"use client";

import Script from "next/script";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState, type ReactNode } from "react";
import { BookingTimeline } from "@/components/features/booking-timeline";
import { ExtendSheet } from "@/components/features/extend-sheet";
import { InvoiceCard } from "@/components/features/invoice-card";
import { PaymentPanel } from "@/components/features/payment-panel";
import { RefundPanel } from "@/components/features/refund-panel";
import { ReviewSheet } from "@/components/features/review-sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { ConfirmDialog } from "@/components/ui/dialog";
import { InlineAlert } from "@/components/ui/inline-alert";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import type { BookingDetail, Invoice, Payment, Refund } from "@/lib/booking";
import { formatDateShort, previousDay } from "@/lib/format";
import { bookingStatusLabel, durationLabel, rentTypeLabel, rentTypeUnit } from "@/lib/labels";
import { openSnap, SNAP_SCRIPT_URL, type SnapPayment } from "@/lib/snap";
import { toast } from "@/lib/toast";

const STATUS_CHECK_INTERVAL_MS = 3000;
const MAX_STATUS_CHECKS = 10;
const CANCELLABLE_STATUSES = ["pending", "paid"];
const PAYABLE_STATUSES = ["pending", "approved"];

export interface BookingDetailClientProps {
  booking: BookingDetail;
  payment: Payment | null;
  refund: Refund | null;
  invoices: Invoice[];
  monthlyDurationOptions?: number[];
}

export function BookingDetailClient({
  booking,
  payment,
  refund,
  invoices,
  monthlyDurationOptions = [],
}: BookingDetailClientProps) {
  const router = useRouter();
  const refresh = useCallback(() => router.refresh(), [router]);
  const navigate = useCallback((href: string) => router.push(href), [router]);
  const [payingPath, setPayingPath] = useState<string | null>(null);
  const [statusCheck, setStatusCheck] = useState<{ from: string; attempts: number } | null>(null);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [extendOpen, setExtendOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);

  const status = bookingStatusLabel(booking.status);
  const statusKey = [booking.status, ...invoices.map((invoice) => invoice.status)].join();
  const checkingStatus =
    statusCheck !== null && statusCheck.from === statusKey && statusCheck.attempts < MAX_STATUS_CHECKS;
  const bookingCheckoutPath = `/api/booking/${booking.id}/checkout`;
  const invoicesPayable = PAYABLE_STATUSES.includes(booking.status);
  const needsPayment = invoicesPayable && invoices.some((invoice) => invoice.status === "unpaid");

  useEffect(() => {
    if (!checkingStatus) return;
    const timer = setTimeout(() => {
      setStatusCheck((current) => current && { ...current, attempts: current.attempts + 1 });
      router.refresh();
    }, STATUS_CHECK_INTERVAL_MS);
    return () => clearTimeout(timer);
  }, [checkingStatus, statusCheck?.attempts, router]);

  async function pay(checkoutPath: string) {
    setPayingPath(checkoutPath);
    const result = await callBff<{ payment: SnapPayment }>(checkoutPath, { method: "POST" });
    setPayingPath(null);
    if (!result.ok) {
      showApiError(result, navigate, () => pay(checkoutPath));
      if (result.httpStatus === 400) refresh();
      return;
    }
    openSnap(result.data.payment, () => setStatusCheck({ from: statusKey, attempts: 0 }));
  }

  async function cancel() {
    setCancelling(true);
    const result = await callBff(`/api/booking/${booking.id}/cancel`, { method: "POST" });
    setCancelling(false);
    setCancelOpen(false);
    if (!result.ok) {
      showApiError(result, navigate, cancel);
      if (result.httpStatus === 400) refresh();
      return;
    }
    toast.success(result.data.meta.message);
    refresh();
  }

  return (
    <div className="flex flex-col gap-6 px-4 py-4">
      {needsPayment && (
        <Script src={SNAP_SCRIPT_URL} data-client-key={process.env.NEXT_PUBLIC_MIDTRANS_CLIENT_KEY} strategy="afterInteractive" />
      )}

      <div className="flex flex-col items-start gap-4">
        <Badge color={status.color}>{status.label}</Badge>
        <BookingTimeline booking={booking} />
      </div>

      {checkingStatus && <InlineAlert variant="info">Memeriksa status pembayaran…</InlineAlert>}

      {booking.status === "pending" && (
        <PaymentPanel
          expiresAt={booking.expires_at}
          activePaymentExpiresAt={payment?.status === "pending" ? payment.expires_at : undefined}
          onExpire={refresh}
          paying={payingPath === bookingCheckoutPath}
          onPay={() => pay(bookingCheckoutPath)}
        />
      )}

      {refund && <RefundPanel refund={refund} bookingId={booking.id} onSaved={refresh} />}

      <section className="flex flex-col gap-3">
        <h2 className="text-title-sm font-semibold text-neutral-900">Rincian sewa</h2>
        <dl className="flex flex-col gap-2">
          <DetailRow label="Kamar">{booking.room.name}</DetailRow>
          <DetailRow label="Kost">{booking.property.name}</DetailRow>
          <DetailRow label="Tipe sewa">{rentTypeLabel(booking.rent_type)}</DetailRow>
          <DetailRow label="Durasi">{durationLabel(booking.rent_type, booking.duration)}</DetailRow>
          <DetailRow label="Mulai">{formatDateShort(booking.start_date)}</DetailRow>
          <DetailRow label="Selesai">{formatDateShort(previousDay(booking.end_date))}</DetailRow>
          <DetailRow label="Harga sewa">
            {booking.price_label} /{rentTypeUnit(booking.rent_type)}
          </DetailRow>
          <DetailRow label="Deposit">{booking.deposit_label}</DetailRow>
          {booking.admin_fee > 0 && <DetailRow label="Biaya admin">{booking.admin_fee_label}</DetailRow>}
          <DetailRow label="Pembayaran pertama">{booking.total_payment_label}</DetailRow>
        </dl>
      </section>

      {invoices.length > 0 && (
        <section className="flex flex-col gap-3">
          <h2 className="text-title-sm font-semibold text-neutral-900">Tagihan</h2>
          {invoices.map((invoice) => {
            const checkoutPath = `/api/invoices/${invoice.id}/checkout`;
            return (
              <InvoiceCard
                key={invoice.id}
                invoice={invoice}
                paying={payingPath === checkoutPath}
                onPay={invoicesPayable ? () => pay(checkoutPath) : undefined}
              />
            );
          })}
        </section>
      )}

      {booking.can_extend && booking.can_review ? (
        <ButtonPair>
          <Button onClick={() => setExtendOpen(true)}>
            Perpanjang sewa
          </Button>
          <Button onClick={() => setReviewOpen(true)}>
            Tulis ulasan
          </Button>
        </ButtonPair>
      ) : booking.can_extend ? (
        <Button fullWidth onClick={() => setExtendOpen(true)}>
          Perpanjang sewa
        </Button>
      ) : (
        booking.can_review && (
          <Button fullWidth onClick={() => setReviewOpen(true)}>
            Tulis ulasan
          </Button>
        )
      )}

      {CANCELLABLE_STATUSES.includes(booking.status) && (
        <Button
          variant="danger"
          fullWidth
          onClick={() => setCancelOpen(true)}
        >
          Batalkan booking
        </Button>
      )}

      <ConfirmDialog
        open={cancelOpen}
        onClose={() => setCancelOpen(false)}
        title="Batalkan booking?"
        description={cancelConsequence(booking)}
        confirmLabel="Batalkan"
        cancelLabel="Tidak"
        onConfirm={cancel}
        loading={cancelling}
      />

      {booking.can_review && (
        <ReviewSheet
          open={reviewOpen}
          onClose={() => setReviewOpen(false)}
          bookingId={booking.id}
          navigate={navigate}
          onReviewed={refresh}
        />
      )}

      {booking.can_extend && (
        <ExtendSheet
          open={extendOpen}
          onClose={() => setExtendOpen(false)}
          bookingId={booking.id}
          rentType={booking.rent_type}
          startDate={booking.start_date}
          currentDuration={booking.duration}
          monthlyOptions={monthlyDurationOptions}
          navigate={navigate}
          onExtended={refresh}
        />
      )}
    </div>
  );
}

function cancelConsequence(booking: BookingDetail): string {
  if (booking.status !== "paid") return "Pengajuan sewa ini dibatalkan dan tidak bisa dipulihkan.";
  if (booking.admin_fee > 0) {
    return `Dana yang sudah kamu bayar dikembalikan setelah dikurangi biaya admin ${booking.admin_fee_label}.`;
  }
  return "Dana yang sudah kamu bayar dikembalikan penuh.";
}

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="shrink-0 text-body text-neutral-600">{label}</dt>
      <dd className="min-w-0 break-words text-right text-body tabular-nums text-neutral-900">{children}</dd>
    </div>
  );
}
