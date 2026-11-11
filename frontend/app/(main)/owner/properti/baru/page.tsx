import { AppHeader } from "@/components/features/app-header";
import { OwnerPropertyForm, type AreaOption } from "@/components/features/owner-property-form";
import { ForbiddenState } from "@/components/ui/error-state";
import { InlineAlert } from "@/components/ui/inline-alert";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";

export const metadata = { title: "Tambah properti" };

export default async function NewOwnerPropertyPage() {
  await requireAuthenticated("/owner/properti/baru");
  const [role, areas] = await Promise.all([getNavRole(), apiFetch<{ areas: AreaOption[] }>("/area/city", { auth: false })]);

  return (
    <>
      <AppHeader variant="inner" title="Tambah properti" role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : (
        <main className="mx-auto flex max-w-[640px] flex-col gap-4 px-4 py-4">
          <InlineAlert variant="info">Properti baru ditinjau admin sebelum tampil di pencarian.</InlineAlert>
          <OwnerPropertyForm areas={areas.ok ? areas.data.areas : []} />
        </main>
      )}
    </>
  );
}
