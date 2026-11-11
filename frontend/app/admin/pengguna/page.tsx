import { AdminUserList } from "@/components/features/admin-user-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE, type AdminUser } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";

export const metadata = { title: "Pengguna" };

export default async function AdminUsersPage() {
  const result = await apiFetch<{ users: AdminUser[] }>("/admin/users", { searchParams: { limit: ADMIN_PAGE_SIZE } });
  return result.ok ? <AdminUserList initial={result.data.users} /> : <ErrorState title={result.meta.message} />;
}
