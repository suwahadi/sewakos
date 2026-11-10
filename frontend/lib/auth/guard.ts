import { redirect } from "next/navigation";
import { getSessionToken } from "./session";

/**
 * DESIGN.md §7.0 — guests are redirected to /masuk?next={path}. Role-based guarding ("Role yang
 * salah → ErrorState 403") needs the account's actual role, which isn't retrievable yet without
 * B2.2's `GET /me` — pages needing it should render `ForbiddenState` (components/ui/error-state)
 * themselves once that's wired up.
 */
export async function requireAuthenticated(currentPath: string): Promise<void> {
  const token = await getSessionToken();
  if (!token) {
    redirect(`/masuk?next=${encodeURIComponent(currentPath)}`);
  }
}
