import Link from "next/link";
import { AppHeader } from "@/components/features/app-header";
import { OwnerPropertyCard } from "@/components/features/owner-property-card";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import type { OwnerProperty } from "@/lib/owner";

export const metadata = { title: "Properti" };

export default async function OwnerPropertiesPage() {
  await requireAuthenticated("/owner/properti");
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<{ properties: OwnerProperty[] }>("/owner/properties", { searchParams: { limit: 50 } }),
  ]);

  if (role !== "owner" || !result.ok) {
    return (
      <>
        <AppHeader variant="inner" title="Properti" role={role} />
        {role !== "owner" ? <ForbiddenState /> : <ErrorState title={result.ok ? "" : result.meta.message} />}
      </>
    );
  }

  const properties = result.data.properties;
  return (
    <>
      <AppHeader variant="inner" title="Properti" role={role} />
      <main className="mx-auto flex max-w-[1200px] flex-col gap-4 px-4 py-4">
        <Link
          href="/owner/properti/baru"
          className="inline-flex h-11 w-full items-center justify-center rounded-md bg-brand-600 px-4 text-body-strong text-neutral-0 hover:bg-brand-700 sm:w-auto sm:self-start"
        >
          Tambah properti
        </Link>
        {properties.length > 0 ? (
          <div className="grid gap-3 lg:grid-cols-2">
            {properties.map((property) => (
              <OwnerPropertyCard key={property.id} property={property} />
            ))}
          </div>
        ) : (
          <p className="text-body text-neutral-600">Tambahkan kost pertamamu agar bisa tampil di pencarian.</p>
        )}
      </main>
    </>
  );
}
