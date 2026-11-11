import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { SewaClient, type DraftPriceItem } from "@/components/features/sewa-client";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";

export const metadata = { title: "Ajukan sewa" };

interface ListingResponse {
  property: { id: number; name: string; is_flash_sale: boolean };
  rooms: { id: number; name: string; monthly_duration_options: number[] }[];
}

export default async function SewaPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ room?: string }>;
}) {
  const { id } = await params;
  const { room: roomParam } = await searchParams;
  await requireAuthenticated(`/kost/${id}/sewa${roomParam ? `?room=${encodeURIComponent(roomParam)}` : ""}`);

  const [role, listing] = await Promise.all([getNavRole(), apiFetch<ListingResponse>(`/listings/${id}`)]);
  if (role === "owner") {
    return (
      <>
        <AppHeader variant="inner" title="Ajukan sewa" role={role} />
        <ForbiddenState />
      </>
    );
  }
  if (!listing.ok) notFound();
  const { property, rooms } = listing.data;
  const room = rooms.find((candidate) => String(candidate.id) === roomParam);
  if (!room) notFound();

  const prices = await apiFetch<{ data: DraftPriceItem[] }>(`/booking/draft-price/${room.id}`, {
    auth: false,
    searchParams: { is_flash_sale: property.is_flash_sale, duration: 1 },
  });

  return (
    <>
      <AppHeader variant="inner" title="Ajukan sewa" role={role} />
      <div className="px-4 py-4">
        <h1 className="text-title-sm font-semibold text-neutral-900">{room.name}</h1>
        <p className="text-caption text-neutral-600">{property.name}</p>
      </div>
      {prices.ok ? (
        <SewaClient
          roomId={room.id}
          isFlashSale={property.is_flash_sale}
          monthlyDurationOptions={room.monthly_duration_options}
          initialPrices={prices.data.data}
        />
      ) : (
        <ErrorState title={prices.meta.message} />
      )}
    </>
  );
}
