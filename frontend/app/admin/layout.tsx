import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { AdminNav } from "@/components/features/admin-nav";
import { ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";

export const metadata = { title: { default: "Admin", template: "%s | Admin Sewa Kos" } };

export default async function AdminLayout({ children }: { children: ReactNode }) {
  await requireAuthenticated("/admin");
  const me = await apiFetch<{ user: { role: string } }>("/me");
  if (!me.ok) redirect("/masuk?next=%2Fadmin");
  if (me.data.user.role !== "admin") {
    return (
      <main className="flex min-h-dvh items-center justify-center px-4">
        <ForbiddenState />
      </main>
    );
  }

  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[240px_minmax(0,1fr)]">
      <AdminNav />
      <main className="mx-auto w-full min-w-0 max-w-[1200px] px-4 py-6 lg:px-8">{children}</main>
    </div>
  );
}
