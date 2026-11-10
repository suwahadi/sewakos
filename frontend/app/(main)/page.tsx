import { AppHeader } from "@/components/features/app-header";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";
import { getSessionToken } from "@/lib/auth/session";

async function getUnreadCount(): Promise<number> {
  const token = await getSessionToken();
  if (!token) return 0;
  const result = await apiFetch<{ unread: number }>("/notifications/counter");
  return result.ok ? result.data.unread : 0;
}

export default async function HomePage() {
  const [role, unreadCount] = await Promise.all([getNavRole(), getUnreadCount()]);

  return (
    <>
      <AppHeader variant="home" role={role} unreadCount={unreadCount} />
      <main className="mx-auto max-w-[1200px] px-4 py-8">
        <h1 className="text-display font-bold text-neutral-900">Cari kost yang pas</h1>
      </main>
    </>
  );
}
