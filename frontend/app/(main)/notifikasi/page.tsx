import { AppHeader } from "@/components/features/app-header";
import { NotifikasiClient } from "@/components/features/notifikasi-client";
import type { Notification } from "@/components/features/notification-item";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";

interface NotificationListResponse {
  notifications: Notification[];
  current_page: number;
}

export default async function NotifikasiPage() {
  await requireAuthenticated("/notifikasi");
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<NotificationListResponse>("/notifications", { searchParams: { page: "1", limit: "20" } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Notifikasi" role={role} />
      <NotifikasiClient initial={result.ok ? result.data.notifications : []} />
    </>
  );
}
