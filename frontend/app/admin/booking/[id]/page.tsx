import Link from "next/link";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import { ErrorState } from "@/components/ui/error-state";
import type { AdminBooking } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";
import { formatDateShort, formatPhoneDisplay, formatTimeWIB, previousDay } from "@/lib/format";
import { bookingStatusLabel, durationLabel, payoutStatusLabel, paymentStatusLabel, refundStatusLabel } from "@/lib/labels";

export const metadata = { title: "Detail booking" };

interface AdminBookingDetail {
  booking: AdminBooking;
  payment: { order_id: string; status: string; expires_at: string } | null;
  refund: { amount_label: string; status: string; completed_at: string | null } | null;
  payout: { amount_label: string; status: string; transfer_reference: string | null; completed_at: string | null } | null;
}

function dateTime(value: string): string {
  return `${formatDateShort(value)}, ${formatTimeWIB(value)}`;
}

export default async function AdminBookingPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const result = await apiFetch<AdminBookingDetail>(`/admin/bookings/${encodeURIComponent(id)}`);
  if (!result.ok && result.httpStatus === 404) notFound();
  if (!result.ok) return <ErrorState title={result.meta.message} />;
  const { booking, payment, refund, payout } = result.data;
  const status = bookingStatusLabel(booking.status);

  return (
    <div className="flex flex-col gap-6">
      <Link href="/admin/booking" className="text-label font-semibold text-brand-700">
        Kembali ke daftar
      </Link>
      <div className="flex flex-col items-start gap-2">
        <Badge color={status.color}>{status.label}</Badge>
        <h1 className="text-title-lg font-semibold text-neutral-900">Booking #{booking.id}</h1>
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Panel title="Sewa">
          <Row label="Penyewa">{booking.tenant.name}</Row>
          <Row label="Nomor HP">{booking.tenant.phone_number ? formatPhoneDisplay(booking.tenant.phone_number) : "–"}</Row>
          <Row label="Kamar">{booking.room.name}</Row>
          <Row label="Kost">{booking.property.name}</Row>
          <Row label="Durasi">{durationLabel(booking.rent_type, booking.duration)}</Row>
          <Row label="Periode">
            {formatDateShort(booking.start_date)} – {formatDateShort(previousDay(booking.end_date))}
          </Row>
          {booking.reject_reason && <Row label="Alasan ditolak">{booking.reject_reason}</Row>}
        </Panel>
        <Panel title="Uang">
          <Row label="Harga sewa">{booking.price_label}</Row>
          <Row label="Deposit">{booking.deposit_label}</Row>
          <Row label="Biaya admin">{booking.admin_fee_label}</Row>
          <Row label="Dana pemilik">{booking.payout_amount_label}</Row>
          {booking.paid_at && <Row label="Dibayar">{dateTime(booking.paid_at)}</Row>}
          {booking.owner_deadline_at && booking.status === "paid" && <Row label="Batas respons">{dateTime(booking.owner_deadline_at)}</Row>}
        </Panel>
        <Panel title="Pembayaran Midtrans">
          {payment ? (
            <>
              <Row label="Order ID">{payment.order_id}</Row>
              <Row label="Status">{paymentStatusLabel(payment.status).label}</Row>
              <Row label="Kedaluwarsa">{dateTime(payment.expires_at)}</Row>
            </>
          ) : (
            <p className="text-body text-neutral-600">Belum ada transaksi.</p>
          )}
        </Panel>
        <Panel title="Refund dan payout">
          <Row label="Refund">{refund ? `${refund.amount_label}, ${refundStatusLabel(refund.status).label}` : "Tidak ada"}</Row>
          <Row label="Payout">{payout ? `${payout.amount_label}, ${payoutStatusLabel(payout.status).label}` : "Belum ada"}</Row>
          {payout?.transfer_reference && <Row label="Referensi payout">{payout.transfer_reference}</Row>}
        </Panel>
      </div>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2 rounded-lg border border-neutral-200 p-4">
      <h2 className="text-title-sm font-semibold text-neutral-900">{title}</h2>
      <dl className="flex flex-col gap-2">{children}</dl>
    </section>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="shrink-0 text-body text-neutral-600">{label}</dt>
      <dd className="min-w-0 break-words text-right text-body text-neutral-900">{children}</dd>
    </div>
  );
}
