import { AppHeader } from "@/components/features/app-header";
import { BookingListClient } from "@/components/features/booking-list-client";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import { BOOKING_PAGE_SIZE, type BookingSummary } from "@/lib/booking";

export const metadata = { title: "Booking" };

export default async function BookingPage() {
  await requireAuthenticated("/booking");
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<{ bookings: BookingSummary[] }>("/booking/my", { searchParams: { page: 1, limit: BOOKING_PAGE_SIZE } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Booking" role={role} />
      {role === "owner" ? (
        <ForbiddenState />
      ) : result.ok ? (
        <BookingListClient initial={result.data.bookings} />
      ) : (
        <ErrorState title={result.meta.message} />
      )}
    </>
  );
}
