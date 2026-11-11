import { AdminReconciliationList } from "@/components/features/admin-finance";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminReconciliationIssue } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Rekonsiliasi" };

export default async function AdminReconciliationPage() {
  const result = await apiFetch<{ issues: AdminReconciliationIssue[] }>("/admin/finance/reconciliation", {
    searchParams: { status: "open", limit: ADMIN_PAGE_SIZE },
  });
  return result.ok ? <AdminReconciliationList initial={result.data.issues} /> : <ErrorState title={result.meta.message} />;
}
