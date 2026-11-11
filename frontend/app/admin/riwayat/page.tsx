import { AdminAuditLogList } from "@/components/features/admin-audit-log-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminAuditLogEntry } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Riwayat aktivitas" };

export default async function AdminAuditLogPage() {
  const result = await apiFetch<{ audit_logs: AdminAuditLogEntry[] }>("/admin/audit-logs", {
    searchParams: { limit: ADMIN_PAGE_SIZE },
  });

  return result.ok ? <AdminAuditLogList initialEntries={result.data.audit_logs} /> : <ErrorState title={result.meta.message} />;
}
