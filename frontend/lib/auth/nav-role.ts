import { cache } from "react";
import type { NavRole } from "@/components/features/nav-items";
import { apiFetch } from "@/lib/api/client";
import { getSessionRole, getSessionToken } from "./session";

interface MeResponseData {
  user: { id: number; name: string; role: string };
}

// Admin has no bottom-nav/app-shell presence yet (its panel is F5, a separate layout) — default
// it to the tenant item set rather than adding a third branch with nowhere real to point.
export const getNavRole = cache(async (): Promise<NavRole> => {
  const token = await getSessionToken();
  if (!token) return "guest";

  const storedRole = await getSessionRole();
  if (storedRole) return storedRole === "owner" ? "owner" : "tenant";

  const result = await apiFetch<MeResponseData>("/me");
  if (!result.ok) return "guest";
  return result.data.user.role === "owner" ? "owner" : "tenant";
});
