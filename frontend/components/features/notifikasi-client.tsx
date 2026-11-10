"use client";

import { Bell } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { EmptyState } from "@/components/ui/empty-state";
import { LoadMore } from "@/components/ui/load-more";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import { callBff } from "@/lib/api/browser";
import { toast } from "@/lib/toast";
import { Notification, NotificationItem } from "./notification-item";

type Category = "all" | Notification["category"];

const TABS: TabItem<Category>[] = [
  { value: "all", label: "Semua" },
  { value: "booking", label: "Booking" },
  { value: "payment", label: "Pembayaran" },
  { value: "promo", label: "Promo" },
];

interface NotificationListResponse {
  notifications: Notification[];
  current_page: number;
}

export function NotifikasiClient({ initial }: { initial: Notification[] }) {
  const router = useRouter();
  const [category, setCategory] = useState<Category>("all");
  const [items, setItems] = useState(initial);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= 20);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadingCategory, setLoadingCategory] = useState(false);

  async function loadCategory(next: Category) {
    setCategory(next);
    setLoadingCategory(true);
    const result = await callBff<NotificationListResponse>(
      `/api/notifications?${next === "all" ? "" : `category=${next}&`}page=1&limit=20`,
    );
    setLoadingCategory(false);
    if (!result.ok) {
      toast.error(result.meta.message);
      return;
    }
    setItems(result.data.notifications);
    setPage(1);
    setHasMore(result.data.notifications.length >= 20);
  }

  async function loadMore() {
    setLoadingMore(true);
    const nextPage = page + 1;
    const result = await callBff<NotificationListResponse>(
      `/api/notifications?${category === "all" ? "" : `category=${category}&`}page=${nextPage}&limit=20`,
    );
    setLoadingMore(false);
    if (!result.ok) {
      toast.error(result.meta.message);
      return;
    }
    setItems((previous) => [...previous, ...result.data.notifications]);
    setPage(nextPage);
    setHasMore(result.data.notifications.length >= 20);
  }

  async function handleClick(notification: Notification) {
    if (notification.read_at === null) {
      setItems((previous) =>
        previous.map((item) => (item.id === notification.id ? { ...item, read_at: new Date().toISOString() } : item)),
      );
      callBff(`/api/notifications/${notification.id}/read`, { method: "PATCH" });
    }
    if (notification.link) router.push(notification.link);
  }

  return (
    <div className="flex flex-col">
      <div className="border-b border-neutral-200 px-4">
        <Tabs items={TABS} value={category} onChange={loadCategory} aria-label="Kategori notifikasi" idPrefix="notifikasi" />
      </div>
      {loadingCategory ? null : items.length === 0 ? (
        <EmptyState icon={Bell} title="Belum ada notifikasi" description="Notifikasi kamu akan muncul di sini." />
      ) : (
        <>
          <div className="flex flex-col divide-y divide-neutral-200">
            {items.map((notification) => (
              <NotificationItem key={notification.id} notification={notification} onClick={() => handleClick(notification)} />
            ))}
          </div>
          {hasMore && (
            <div className="p-4">
              <LoadMore loading={loadingMore} onClick={loadMore} />
            </div>
          )}
        </>
      )}
    </div>
  );
}
