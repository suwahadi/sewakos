import { AdminPropertyList } from "@/components/features/admin-property-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminProperty } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Moderasi properti" };

export default async function AdminPropertiesPage() {
  const result = await apiFetch<{ properties: AdminProperty[] }>("/admin/properties", {
    searchParams: { moderation_status: "pending", limit: ADMIN_PAGE_SIZE },
  });
  return result.ok ? <AdminPropertyList initial={result.data.properties} /> : <ErrorState title={result.meta.message} />;
}
