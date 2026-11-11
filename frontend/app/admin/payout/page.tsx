import { AdminPayoutList } from "@/components/features/admin-transfers";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminPayout } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Payout" };

export default async function AdminPayoutsPage() {
  const result = await apiFetch<{ payouts: AdminPayout[] }>("/admin/payouts", {
    searchParams: { status: "pending", limit: ADMIN_PAGE_SIZE },
  });
  return result.ok ? <AdminPayoutList initial={result.data.payouts} /> : <ErrorState title={result.meta.message} />;
}
