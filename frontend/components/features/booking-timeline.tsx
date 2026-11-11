import { CircleCheck, CircleX } from "lucide-react";
import type { BookingDetail } from "@/lib/booking";
import { cn } from "@/lib/cn";
import { bookingStatusLabel } from "@/lib/labels";

const STEPS = ["Diajukan", "Dibayar", "Disetujui pemilik", "Sewa selesai"];
const STOPPED_STATUSES = ["rejected", "cancelled", "terminated"];
const APPROVED_STATUSES = ["approved", "completed", "terminated"];

function completedStepCount(booking: Pick<BookingDetail, "status" | "paid_at">): number {
  if (booking.status === "completed") return 4;
  if (APPROVED_STATUSES.includes(booking.status)) return 3;
  return booking.paid_at ? 2 : 1;
}

export function BookingTimeline({ booking }: { booking: Pick<BookingDetail, "status" | "paid_at" | "reject_reason"> }) {
  const completed = completedStepCount(booking);
  const stopped = STOPPED_STATUSES.includes(booking.status);
  const steps = stopped ? STEPS.slice(0, completed) : STEPS;

  return (
    <ol aria-label="Status booking" className="flex flex-col gap-3">
      {steps.map((label, index) => {
        const done = index < completed;
        const active = index === completed;
        return (
          <li key={label} aria-current={active ? "step" : undefined} className="flex items-center gap-3">
            <span className="flex size-5 shrink-0 items-center justify-center">
              {done ? (
                <CircleCheck aria-hidden="true" size={20} strokeWidth={1.75} className="text-success" />
              ) : (
                <span className={cn("size-2.5 rounded-full", active ? "bg-brand-600" : "bg-neutral-300")} />
              )}
            </span>
            <span className={cn("text-body", active ? "font-semibold text-neutral-900" : "text-neutral-700")}>{label}</span>
          </li>
        );
      })}
      {stopped && (
        <li className="flex items-start gap-3 text-danger">
          <CircleX aria-hidden="true" size={20} strokeWidth={1.75} className="mt-0.5 shrink-0" />
          <div className="min-w-0">
            <p className="text-body font-semibold">{bookingStatusLabel(booking.status).label}</p>
            {booking.reject_reason && <p className="break-words text-body">{booking.reject_reason}</p>}
          </div>
        </li>
      )}
    </ol>
  );
}
