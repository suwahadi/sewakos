import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { OwnerBookingDetail } from "@/components/features/owner-booking-detail";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import type { OwnerBooking } from "@/lib/owner";

export const metadata = { title: "Detail booking masuk" };

export default async function OwnerBookingDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  await requireAuthenticated(`/owner/booking/${encodeURIComponent(id)}`);
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<{ booking: OwnerBooking }>(`/owner/bookings/${encodeURIComponent(id)}`),
  ]);

  if (role === "owner" && !result.ok && result.httpStatus === 404) notFound();
  return (
    <>
      <AppHeader variant="inner" title={result.ok ? `Booking #${result.data.booking.id}` : "Booking"} role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : result.ok ? (
        <OwnerBookingDetail booking={result.data.booking} />
      ) : (
        <ErrorState title={result.meta.message} />
      )}
    </>
  );
}
