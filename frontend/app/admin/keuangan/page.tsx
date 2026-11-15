import { AdminFinanceSummaryAndLedger } from "@/components/features/admin-finance";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminFinanceSummary, type AdminLedgerEntry, type AdminUser } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Keuangan" };

function parseOwnerId(value: string | undefined): number | null {
  if (!value || !/^\d+$/.test(value)) return null;
  const id = Number(value);
  return id > 0 ? id : null;
}

export default async function AdminFinancePage({ searchParams }: { searchParams: Promise<{ user?: string }> }) {
  const ownerId = parseOwnerId((await searchParams).user);
  const ownerFilter = ownerId ? { owner_id: String(ownerId) } : {};

  const [summary, ledger, owner] = await Promise.all([
    apiFetch<{ summary: AdminFinanceSummary }>("/admin/finance/summary", { searchParams: ownerFilter }),
    apiFetch<{ entries: AdminLedgerEntry[] }>("/admin/finance/ledger", { searchParams: { ...ownerFilter, limit: ADMIN_PAGE_SIZE } }),
    ownerId ? apiFetch<{ user: AdminUser }>(`/admin/users/${ownerId}`) : Promise.resolve(null),
  ]);

  if (!ledger.ok) return <ErrorState title={ledger.meta.message} />;

  const selectedOwner = ownerId ? { id: ownerId, name: owner?.ok ? owner.data.user.name : `Pengguna #${ownerId}` } : null;
  return (
    <AdminFinanceSummaryAndLedger
      key={ownerId ?? "all"}
      summary={summary.ok ? summary.data.summary : null}
      initialLedger={ledger.data.entries}
      owner={selectedOwner}
    />
  );
}
