import type { ReactNode } from "react";
import { BottomNav } from "@/components/features/bottom-nav";
import { MainShell } from "@/components/features/main-shell";
import { getNavRole } from "@/lib/auth/nav-role";

// AppHeader isn't rendered here: its "home" vs "inner" variant (and "inner"'s title/rightAction,
// DESIGN.md §6.22) differ per page, so each page under this group renders its own AppHeader —
// this layout only owns BottomNav (same for every page) and the page-bottom padding that makes
// room for it. Belongs in layout.tsx rather than template.tsx (§5.1) so BottomNav doesn't replay
// the page-enter animation on every navigation.
export default async function MainLayout({ children }: { children: ReactNode }) {
  const role = await getNavRole();
  return (
    <>
      <MainShell>{children}</MainShell>
      <BottomNav role={role} />
    </>
  );
}
