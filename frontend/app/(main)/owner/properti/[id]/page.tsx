import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { OwnerPropertyDetail } from "@/components/features/owner-property-detail";
import { PhotoUploader } from "@/components/features/photo-uploader";
import type { AreaOption } from "@/components/features/owner-property-form";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import type { OwnerProperty, OwnerRoom } from "@/lib/owner";

export const metadata = { title: "Kelola properti" };

export default async function OwnerPropertyPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const path = `/owner/properties/${encodeURIComponent(id)}`;
  await requireAuthenticated(`/owner/properti/${encodeURIComponent(id)}`);
  const [role, property, rooms, areas] = await Promise.all([
    getNavRole(),
    apiFetch<{ property: OwnerProperty }>(path),
    apiFetch<{ rooms: OwnerRoom[] }>(`${path}/rooms`),
    apiFetch<{ areas: AreaOption[] }>("/area/city", { auth: false }),
  ]);

  if (role === "owner" && !property.ok && property.httpStatus === 404) notFound();
  return (
    <>
      <AppHeader variant="inner" title={property.ok ? property.data.property.name : "Properti"} role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : !property.ok ? (
        <ErrorState title={property.meta.message} />
      ) : (
        <OwnerPropertyDetail
          property={property.data.property}
          rooms={rooms.ok ? rooms.data.rooms : []}
          areas={areas.ok ? areas.data.areas : []}
          photos={<PhotoUploader propertyId={property.data.property.id} initialPhotos={property.data.property.photos} />}
        />
      )}
    </>
  );
}
