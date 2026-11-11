"use client";

import { ListChecks } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { OwnerBookingCard } from "@/components/features/owner-booking-card";
import { EmptyState } from "@/components/ui/empty-state";
import { LoadMore } from "@/components/ui/load-more";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { OWNER_PAGE_SIZE, type OwnerBooking } from "@/lib/owner";

type StatusTab = "all" | "paid" | "approved" | "completed";

const TABS: TabItem<StatusTab>[] = [
  { value: "all", label: "Semua" },
  { value: "paid", label: "Perlu direspons" },
  { value: "approved", label: "Aktif" },
  { value: "completed", label: "Selesai" },
];

export function OwnerBookingList({ initial }: { initial: OwnerBooking[] }) {
  const router = useRouter();
  const [status, setStatus] = useState<StatusTab>("all");
  const [bookings, setBookings] = useState(initial);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= OWNER_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function load(nextStatus: StatusTab, nextPage: number) {
    setLoading(true);
    const query = `${nextStatus === "all" ? "" : `status=${nextStatus}&`}page=${nextPage}&limit=${OWNER_PAGE_SIZE}`;
    const result = await callBff<{ bookings: OwnerBooking[] }>(`/api/owner/bookings?${query}`);
    setLoading(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), () => load(nextStatus, nextPage));
      return;
    }
    setBookings((current) => (nextPage === 1 ? result.data.bookings : [...current, ...result.data.bookings]));
    setPage(nextPage);
    setHasMore(result.data.bookings.length >= OWNER_PAGE_SIZE);
  }

  function changeStatus(next: StatusTab) {
    setStatus(next);
    load(next, 1);
  }

  return (
    <div className="flex flex-col">
      <div className="border-b border-neutral-200 px-4">
        <Tabs items={TABS} value={status} onChange={changeStatus} aria-label="Status booking" idPrefix="owner-booking" />
      </div>
      {bookings.length === 0 && !loading ? (
        <EmptyState icon={ListChecks} title="Belum ada booking" description="Booking yang sudah dibayar penyewa akan muncul di sini." />
      ) : (
        <div className="flex flex-col gap-3 px-4 py-4">
          {bookings.map((booking) => (
            <OwnerBookingCard key={booking.id} booking={booking} />
          ))}
          {hasMore && <LoadMore loading={loading} onClick={() => load(status, page + 1)} />}
        </div>
      )}
    </div>
  );
}
