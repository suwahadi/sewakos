import { AdminAuditLogList } from "@/components/features/admin-audit-log-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminAuditLogEntry, type AdminAuditLogFilterOptions } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Riwayat aktivitas" };

export default async function AdminAuditLogPage() {
  const [result, options] = await Promise.all([
    apiFetch<{ audit_logs: AdminAuditLogEntry[] }>("/admin/audit-logs", { searchParams: { limit: ADMIN_PAGE_SIZE } }),
    apiFetch<AdminAuditLogFilterOptions>("/admin/audit-logs/filters"),
  ]);

  if (!result.ok) return <ErrorState title={result.meta.message} />;
  return (
    <AdminAuditLogList
      initialEntries={result.data.audit_logs}
      options={options.ok ? { actors: options.data.actors, actions: options.data.actions } : { actors: [], actions: [] }}
    />
  );
}
