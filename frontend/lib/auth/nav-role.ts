import { cache } from "react";
import type { NavRole } from "@/components/features/nav-items";
import { apiFetch } from "@/lib/api/client";
import { getSessionRole, getSessionToken } from "./session";

interface MeResponseData {
  user: { id: number; name: string; role: string };
}

export class NavRoleUnavailableError extends Error {
  constructor() {
    super("Peran akun belum bisa dimuat.");
    this.name = "NavRoleUnavailableError";
  }
}

// Only a rejected token is a guest; other failures throw so they never masquerade as a guest.
export const getNavRole = cache(async (): Promise<NavRole> => {
  const token = await getSessionToken();
  if (!token) return "guest";

  const storedRole = await getSessionRole();
  if (storedRole) return storedRole === "owner" ? "owner" : "tenant";

  const result = await apiFetch<MeResponseData>("/me");
  if (result.ok) return result.data.user.role === "owner" ? "owner" : "tenant";
  if (result.httpStatus === 401) return "guest";
  throw new NavRoleUnavailableError();
});
