"use client";

import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/cn";
import { formatDateShort, previousDay } from "@/lib/format";
import { bookingStatusLabel } from "@/lib/labels";
import type { OwnerBooking } from "@/lib/owner";
import { formatRemaining, HOUR, useRemainingMs } from "@/lib/use-remaining";

const URGENT_BELOW_MS = 2 * HOUR;

export function ResponseDeadline({ deadline }: { deadline: string }) {
  const remaining = useRemainingMs(new Date(deadline).getTime());
  const urgent = remaining < URGENT_BELOW_MS;
  return (
    <p className={cn("text-caption", urgent ? "font-semibold text-danger" : "text-neutral-600")}>
      {remaining > 0 ? (
        <>
          Respons dalam{" "}
          <span className="tabular-nums" suppressHydrationWarning>
            {formatRemaining(remaining)}
          </span>
        </>
      ) : (
        "Batas respons sudah lewat"
      )}
    </p>
  );
}

export function OwnerBookingCard({ booking }: { booking: OwnerBooking }) {
  const status = bookingStatusLabel(booking.status);

  return (
    <Link href={`/owner/booking/${booking.id}`} className="flex min-w-0 flex-col gap-2 rounded-lg border border-neutral-200 p-3">
      <div className="flex items-center justify-between gap-2">
        <Badge color={status.color}>{status.label}</Badge>
        <span className="text-caption text-neutral-600">#{booking.id}</span>
      </div>
      <p className="break-words text-title-sm font-semibold text-neutral-900">{booking.tenant.name}</p>
      <p className="break-words text-caption text-neutral-600">
        {booking.room.name}, {booking.property.name}
      </p>
      <p className="text-caption text-neutral-600">
        {formatDateShort(booking.start_date)} – {formatDateShort(previousDay(booking.end_date))}
      </p>
      <div className="flex items-end justify-between gap-3">
        <p className="text-price font-bold tabular-nums text-neutral-900">{booking.payout_amount_label}</p>
        {booking.status === "paid" && booking.owner_deadline_at && <ResponseDeadline deadline={booking.owner_deadline_at} />}
      </div>
    </Link>
  );
}
