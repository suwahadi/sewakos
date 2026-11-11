"use client";

import { useRouter } from "next/navigation";
import { useState, type ReactNode } from "react";
import { ResponseDeadline } from "@/components/features/owner-booking-card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { ConfirmDialog } from "@/components/ui/dialog";
import { InlineAlert } from "@/components/ui/inline-alert";
import { Textarea } from "@/components/ui/textarea";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort, formatPhoneDisplay, formatTimeWIB, previousDay } from "@/lib/format";
import { bookingStatusLabel, durationLabel, rentTypeUnit } from "@/lib/labels";
import type { OwnerBooking } from "@/lib/owner";
import { toast } from "@/lib/toast";

export function OwnerBookingDetail({ booking }: { booking: OwnerBooking }) {
  const router = useRouter();
  const [busy, setBusy] = useState<"approve" | "reject" | null>(null);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [reason, setReason] = useState("");
  const status = bookingStatusLabel(booking.status);

  async function respond(action: "approve" | "reject") {
    setBusy(action);
    const result = await callBff(`/api/owner/bookings/${booking.id}/${action}`, {
      method: "POST",
      body: action === "reject" && reason.trim() ? { reason: reason.trim() } : {},
    });
    setBusy(null);
    setRejectOpen(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), () => respond(action));
      if (result.httpStatus === 400) router.refresh();
      return;
    }
    toast.success(result.data.meta.message);
    router.refresh();
  }

  return (
    <div className="flex flex-col gap-6 px-4 py-4">
      <div className="flex flex-col items-start gap-2">
        <Badge color={status.color}>{status.label}</Badge>
        <h1 className="break-words text-title-lg font-semibold text-neutral-900">{booking.tenant.name}</h1>
        {booking.status === "paid" && booking.owner_deadline_at && <ResponseDeadline deadline={booking.owner_deadline_at} />}
      </div>

      {booking.status === "paid" && (
        <InlineAlert variant="info">
          Setujui atau tolak sebelum {formatDateShort(booking.owner_deadline_at!)}, {formatTimeWIB(booking.owner_deadline_at!)}. Jika tidak direspons,
          booking ditolak otomatis dan dana dikembalikan ke penyewa.
        </InlineAlert>
      )}
      {booking.status === "rejected" && booking.reject_reason && (
        <InlineAlert variant="danger" title="Alasan penolakan">
          <p className="break-words">{booking.reject_reason}</p>
        </InlineAlert>
      )}

      <section className="flex flex-col gap-3">
        <h2 className="text-title-sm font-semibold text-neutral-900">Penyewa</h2>
        <dl className="flex flex-col gap-2">
          <DetailRow label="Nama">{booking.tenant.name}</DetailRow>
          <DetailRow label="Nomor HP">
            {booking.tenant.phone_number ? (
              <a href={`tel:${booking.tenant.phone_number}`} className="text-brand-700 underline">
                {formatPhoneDisplay(booking.tenant.phone_number)}
              </a>
            ) : (
              <span className="text-neutral-600">Tampil setelah disetujui</span>
            )}
          </DetailRow>
        </dl>
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-title-sm font-semibold text-neutral-900">Rincian sewa</h2>
        <dl className="flex flex-col gap-2">
          <DetailRow label="Kamar">{booking.room.name}</DetailRow>
          <DetailRow label="Kost">{booking.property.name}</DetailRow>
          <DetailRow label="Durasi">{durationLabel(booking.rent_type, booking.duration)}</DetailRow>
          <DetailRow label="Mulai">{formatDateShort(booking.start_date)}</DetailRow>
          <DetailRow label="Selesai">{formatDateShort(previousDay(booking.end_date))}</DetailRow>
          <DetailRow label="Harga sewa">
            {booking.price_label} /{rentTypeUnit(booking.rent_type)}
          </DetailRow>
          <DetailRow label="Deposit">{booking.deposit_label}</DetailRow>
          <DetailRow label="Dana untukmu">
            <span className="font-semibold">{booking.payout_amount_label}</span>
          </DetailRow>
          {booking.paid_at && <DetailRow label="Dibayar">{formatDateShort(booking.paid_at)}, {formatTimeWIB(booking.paid_at)}</DetailRow>}
        </dl>
      </section>

      {booking.status === "paid" && (
        <ButtonPair>
          <Button variant="outline" disabled={busy !== null} onClick={() => setRejectOpen(true)}>
            Tolak booking
          </Button>
          <Button loading={busy === "approve"} disabled={busy === "reject"} onClick={() => respond("approve")}>
            Setujui booking
          </Button>
        </ButtonPair>
      )}

      <ConfirmDialog
        open={rejectOpen}
        onClose={() => setRejectOpen(false)}
        title="Tolak booking ini?"
        description={
          <div className="flex flex-col gap-3">
            <p>Kamar dilepas dan seluruh dana dikembalikan ke penyewa.</p>
            <Textarea label="Alasan (opsional)" maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} />
          </div>
        }
        confirmLabel="Tolak booking"
        loading={busy === "reject"}
        onConfirm={() => respond("reject")}
      />
    </div>
  );
}

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="shrink-0 text-body text-neutral-600">{label}</dt>
      <dd className="min-w-0 break-words text-right text-body tabular-nums text-neutral-900">{children}</dd>
    </div>
  );
}
