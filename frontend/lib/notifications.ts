import { apiFetch } from "@/lib/api/client";
import { getSessionToken } from "@/lib/auth/session";

export async function getUnreadCount(): Promise<number> {
  const token = await getSessionToken();
  if (!token) return 0;
  const result = await apiFetch<{ unread: number }>("/notifications/counter");
  return result.ok ? result.data.unread : 0;
}
