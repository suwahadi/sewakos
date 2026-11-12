import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { BookingDetailClient } from "@/components/features/booking-detail-client";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import type { BookingDetail, Invoice, Payment, Refund } from "@/lib/booking";

export const metadata = { title: "Detail booking" };

const INVOICE_LIMIT = 50;

interface BookingDetailResponse {
  booking: BookingDetail;
  payment: Payment | null;
  refund: Refund | null;
}

interface ListingRoomsResponse {
  rooms: { id: number; monthly_duration_options: number[] }[];
}

export default async function BookingDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  await requireAuthenticated(`/booking/${encodeURIComponent(id)}`);
  const [role, detail, invoices] = await Promise.all([
    getNavRole(),
    apiFetch<BookingDetailResponse>(`/booking/${encodeURIComponent(id)}`),
    apiFetch<{ invoices: Invoice[] }>("/invoices", { searchParams: { booking_id: id, limit: INVOICE_LIMIT } }),
  ]);

  if (role === "owner") {
    return (
      <>
        <AppHeader variant="inner" title="Booking" role={role} />
        <ForbiddenState />
      </>
    );
  }

  if (!detail.ok) {
    if (detail.httpStatus === 404) notFound();
    return (
      <>
        <AppHeader variant="inner" title="Booking" role={role} />
        {detail.httpStatus === 403 ? <ForbiddenState /> : <ErrorState title={detail.meta.message} />}
      </>
    );
  }

  const { booking, payment, refund } = detail.data;

  let monthlyDurationOptions: number[] = [];
  if (booking.can_extend && booking.rent_type === "monthly") {
    const listing = await apiFetch<ListingRoomsResponse>(`/listings/${booking.property.id}`);
    monthlyDurationOptions = listing.ok
      ? (listing.data.rooms.find((room) => room.id === booking.room.id)?.monthly_duration_options ?? [])
      : [];
  }

  return (
    <>
      <AppHeader variant="inner" title={`Booking #${booking.id}`} role={role} />
      <BookingDetailClient
        booking={booking}
        payment={payment}
        refund={refund}
        invoices={invoices.ok ? invoices.data.invoices : []}
        monthlyDurationOptions={monthlyDurationOptions}
      />
    </>
  );
}
