import { AdminRefundList } from "@/components/features/admin-transfers";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminRefund } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Refund" };

export default async function AdminRefundsPage() {
  const result = await apiFetch<{ refunds: AdminRefund[] }>("/admin/refunds", {
    searchParams: { status: "ready", limit: ADMIN_PAGE_SIZE },
  });
  return result.ok ? <AdminRefundList initial={result.data.refunds} /> : <ErrorState title={result.meta.message} />;
}
