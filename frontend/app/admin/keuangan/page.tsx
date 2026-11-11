import { AdminFinanceSummaryAndLedger } from "@/components/features/admin-finance";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminFinanceSummary, type AdminLedgerEntry } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Keuangan" };

export default async function AdminFinancePage() {
  const [summary, ledger] = await Promise.all([
    apiFetch<{ summary: AdminFinanceSummary }>("/admin/finance/summary"),
    apiFetch<{ entries: AdminLedgerEntry[] }>("/admin/finance/ledger", { searchParams: { limit: ADMIN_PAGE_SIZE } }),
  ]);

  return ledger.ok ? (
    <AdminFinanceSummaryAndLedger summary={summary.ok ? summary.data.summary : null} initialLedger={ledger.data.entries} />
  ) : (
    <ErrorState title={ledger.meta.message} />
  );
}
