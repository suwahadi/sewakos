import type { NavRole } from "@/components/features/nav-items";
import { apiFetch } from "@/lib/api/client";
import { getSessionToken } from "./session";

interface MeResponseData {
  user: { id: number; name: string; role: string };
}

// Admin has no bottom-nav/app-shell presence yet (its panel is F5, a separate layout) — default
// it to the tenant item set rather than adding a third branch with nowhere real to point.
export async function getNavRole(): Promise<NavRole> {
  const token = await getSessionToken();
  if (!token) return "guest";

  const result = await apiFetch<MeResponseData>("/me");
  if (!result.ok) return "guest";
  return result.data.user.role === "owner" ? "owner" : "tenant";
}
