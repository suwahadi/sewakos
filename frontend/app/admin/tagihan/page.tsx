import { AdminInvoiceList } from "@/components/features/admin-booking-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminInvoice } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Tagihan" };

export default async function AdminInvoicesPage() {
  const result = await apiFetch<{ invoices: AdminInvoice[] }>("/admin/invoices", { searchParams: { limit: ADMIN_PAGE_SIZE } });
  return result.ok ? <AdminInvoiceList initial={result.data.invoices} /> : <ErrorState title={result.meta.message} />;
}
