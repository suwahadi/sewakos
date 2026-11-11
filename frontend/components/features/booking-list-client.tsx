"use client";

import { ReceiptText } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { BookingCard } from "@/components/features/booking-card";
import { EmptyState } from "@/components/ui/empty-state";
import { LoadMore } from "@/components/ui/load-more";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { BOOKING_PAGE_SIZE, type BookingSummary } from "@/lib/booking";

export function BookingListClient({ initial }: { initial: BookingSummary[] }) {
  const router = useRouter();
  const [bookings, setBookings] = useState(initial);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= BOOKING_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    setLoading(true);
    const nextPage = page + 1;
    const result = await callBff<{ bookings: BookingSummary[] }>(
      `/api/booking/my?page=${nextPage}&limit=${BOOKING_PAGE_SIZE}`,
    );
    setLoading(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), loadMore);
      return;
    }
    setBookings((current) => [...current, ...result.data.bookings]);
    setPage(nextPage);
    setHasMore(result.data.bookings.length >= BOOKING_PAGE_SIZE);
  }

  if (bookings.length === 0) {
    return (
      <EmptyState
        icon={ReceiptText}
        title="Belum ada booking"
        description="Kost yang kamu ajukan sewanya akan muncul di sini."
        action={{ label: "Cari kost", onClick: () => router.push("/cari") }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-3 px-4 py-4">
      {bookings.map((booking) => (
        <BookingCard key={booking.id} booking={booking} />
      ))}
      {hasMore && <LoadMore loading={loading} onClick={loadMore} />}
    </div>
  );
}
