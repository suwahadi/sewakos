import { AppHeader } from "@/components/features/app-header";
import { AuditLogList } from "@/components/features/audit-log-list";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import type { AuditLogEntry } from "@/lib/owner";

export const metadata = { title: "Riwayat aktivitas" };

const PAGE_SIZE = 20;

export default async function OwnerAuditLogPage() {
  await requireAuthenticated("/owner/riwayat");
  const [role, logs] = await Promise.all([
    getNavRole(),
    apiFetch<{ audit_logs: AuditLogEntry[] }>("/owner/audit-logs", { searchParams: { limit: PAGE_SIZE } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Riwayat aktivitas" role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : !logs.ok ? (
        <ErrorState title={logs.meta.message} />
      ) : (
        <main className="mx-auto max-w-[960px] px-4 py-4">
          <AuditLogList path="/api/owner/audit-logs" initialEntries={logs.data.audit_logs} />
        </main>
      )}
    </>
  );
}
