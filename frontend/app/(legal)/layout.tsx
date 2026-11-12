import type { ReactNode } from "react";
import { AppHeader } from "@/components/features/app-header";
import { SiteFooter } from "@/components/features/site-footer";
import { getNavRole } from "@/lib/auth/nav-role";
import { getUnreadCount } from "@/lib/notifications";

export default async function LegalLayout({ children }: { children: ReactNode }) {
  const [role, unreadCount] = await Promise.all([getNavRole(), getUnreadCount()]);
  return (
    <div className="flex min-h-dvh flex-col">
      <AppHeader variant="home" role={role} unreadCount={unreadCount} />
      <main className="mx-auto flex w-full max-w-[1200px] flex-col gap-6 px-4 py-8">{children}</main>
      <SiteFooter />
    </div>
  );
}
