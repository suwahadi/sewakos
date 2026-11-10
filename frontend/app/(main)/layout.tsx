import type { ReactNode } from "react";
import { BottomNav } from "@/components/features/bottom-nav";
import { MainShell } from "@/components/features/main-shell";
import { getNavRole } from "@/lib/auth/nav-role";

// A template.tsx here would remount BottomNav on every navigation, replaying its enter animation.
export default async function MainLayout({ children }: { children: ReactNode }) {
  const role = await getNavRole();
  return (
    <>
      <MainShell>{children}</MainShell>
      <BottomNav role={role} />
    </>
  );
}
