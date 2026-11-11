import { AdminBookingList } from "@/components/features/admin-booking-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminBooking } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Booking" };

export default async function AdminBookingsPage() {
  const result = await apiFetch<{ bookings: AdminBooking[] }>("/admin/bookings", { searchParams: { limit: ADMIN_PAGE_SIZE } });
  return result.ok ? <AdminBookingList initial={result.data.bookings} /> : <ErrorState title={result.meta.message} />;
}
