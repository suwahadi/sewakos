import { AdminSettings } from "@/components/features/admin-settings";
import { ErrorState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Pengaturan" };

export default async function AdminSettingsPage() {
  const result = await apiFetch<{ settings: { admin_fee: number } }>("/admin/settings");
  return result.ok ? <AdminSettings adminFee={result.data.settings.admin_fee} /> : <ErrorState title={result.meta.message} />;
}
