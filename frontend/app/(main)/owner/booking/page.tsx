import { AppHeader } from "@/components/features/app-header";
import { OwnerBookingList } from "@/components/features/owner-booking-list";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import { OWNER_PAGE_SIZE, type OwnerBooking } from "@/lib/owner";

export const metadata = { title: "Booking masuk" };

export default async function OwnerBookingPage() {
  await requireAuthenticated("/owner/booking");
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<{ bookings: OwnerBooking[] }>("/owner/bookings", { searchParams: { limit: OWNER_PAGE_SIZE } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Booking masuk" role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : result.ok ? (
        <OwnerBookingList initial={result.data.bookings} />
      ) : (
        <ErrorState title={result.meta.message} />
      )}
    </>
  );
}
