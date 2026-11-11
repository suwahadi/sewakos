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

export default async function BookingDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  await requireAuthenticated(`/booking/${encodeURIComponent(id)}`);
  const [role, detail, invoices] = await Promise.all([
    getNavRole(),
    apiFetch<BookingDetailResponse>(`/booking/${encodeURIComponent(id)}`),
    apiFetch<{ invoices: Invoice[] }>("/invoices", { searchParams: { booking_id: id, limit: INVOICE_LIMIT } }),
  ]);

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

  return (
    <>
      <AppHeader variant="inner" title={`Booking #${booking.id}`} role={role} />
      <BookingDetailClient
        booking={booking}
        payment={payment}
        refund={refund}
        invoices={invoices.ok ? invoices.data.invoices : []}
      />
    </>
  );
}
