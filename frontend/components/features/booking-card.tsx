import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { InlineAlert } from "@/components/ui/inline-alert";
import type { BookingSummary } from "@/lib/booking";
import { formatDateShort, formatTimeWIB, previousDay } from "@/lib/format";
import { bookingStatusLabel, rentTypeUnit } from "@/lib/labels";

export function BookingCard({ booking }: { booking: BookingSummary }) {
  const status = bookingStatusLabel(booking.status);

  return (
    <Link href={`/booking/${booking.id}`} className="flex min-w-0 flex-col gap-2 rounded-lg border border-neutral-200 p-3">
      <div className="flex items-center justify-between gap-2">
        <Badge color={status.color}>{status.label}</Badge>
        <span className="text-caption text-neutral-600">#{booking.id}</span>
      </div>
      <p className="break-words text-title-sm font-semibold text-neutral-900">{booking.room.name}</p>
      <p className="text-caption text-neutral-600">
        {formatDateShort(booking.start_date)} – {formatDateShort(previousDay(booking.end_date))}
      </p>
      <p className="text-price font-bold tabular-nums text-neutral-900">
        {booking.price_label}
        <span className="text-body font-normal text-neutral-600"> /{rentTypeUnit(booking.rent_type)}</span>
      </p>
      {booking.status === "pending" && (
        <InlineAlert variant="warning">
          Bayar sebelum {formatDateShort(booking.expires_at)}, {formatTimeWIB(booking.expires_at)}
        </InlineAlert>
      )}
    </Link>
  );
}
