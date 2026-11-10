import type { NavRole } from "@/components/features/nav-items";
import { getSessionToken } from "./session";

/**
 * TEMPORARY placeholder until B2.2's `GET /me` exists: a session cookie only tells us "logged in
 * or not", not the account's actual role. Authenticated sessions default to "tenant" — NAV_ITEMS'
 * tenant set is what a guest can already reach, so an owner briefly seeing tenant-style nav is a
 * cosmetic bug, whereas defaulting to "guest" would wrongly bounce a real logged-in tenant to
 * /masuk. Replace with a real role lookup once F1 wires up `/me`.
 */
export async function getNavRole(): Promise<NavRole> {
  const token = await getSessionToken();
  return token ? "tenant" : "guest";
}
