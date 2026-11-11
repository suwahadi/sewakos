import { AdminAreaList } from "@/components/features/admin-area-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminArea } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Area" };

export default async function AdminAreasPage() {
  const result = await apiFetch<{ areas: AdminArea[] }>("/admin/areas", { searchParams: { limit: ADMIN_PAGE_SIZE } });
  return result.ok ? <AdminAreaList initial={result.data.areas} /> : <ErrorState title={result.meta.message} />;
}
