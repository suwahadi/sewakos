import type { ReactNode } from "react";
import { AppHeader } from "@/components/features/app-header";
import { BottomNav } from "@/components/features/bottom-nav";
import { getNavRole } from "@/lib/auth/nav-role";

// This group currently only holds the home page. A page needing AppHeader's "inner" variant
// (back button + title, DESIGN.md §6.22) will need its own nested layout.tsx under a deeper
// segment to override the header this layout renders — header and bottom nav belong in
// layout.tsx rather than template.tsx (§5.1) specifically so they don't replay the page-enter
// animation on every navigation.
export default async function MainLayout({ children }: { children: ReactNode }) {
  const role = await getNavRole();
  return (
    <>
      <AppHeader variant="home" role={role} />
      <div className="pb-16 lg:pb-0">{children}</div>
      <BottomNav role={role} />
    </>
  );
}
