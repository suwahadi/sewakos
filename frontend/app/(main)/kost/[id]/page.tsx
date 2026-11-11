import { ExternalLink } from "lucide-react";
import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { KostDetailClient } from "@/components/features/kost-detail-client";
import { PropertyGallery, type GalleryPhoto } from "@/components/features/property-gallery";
import type { RoomOptionData } from "@/components/features/room-option";
import { Badge } from "@/components/ui/badge";
import { GenderBadge } from "@/components/ui/gender-badge";
import { RatingSummary } from "@/components/ui/rating";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";

interface PropertyDetailResponse {
  property: {
    id: number;
    name: string;
    gender: string;
    description: string;
    address: string;
    latitude: number;
    longitude: number;
    is_flash_sale: boolean;
    area: { name: string };
    photos: GalleryPhoto[];
    rating: { average: number | null; count: number };
  };
  rooms: RoomOptionData[];
}

export default async function KostDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<PropertyDetailResponse>(`/listings/${encodeURIComponent(id)}`),
  ]);

  if (!result.ok) notFound();
  const { property, rooms } = result.data;
  const mapsUrl = `https://www.google.com/maps/search/?api=1&query=${property.latitude},${property.longitude}`;

  return (
    <>
      <AppHeader variant="inner" title={property.name} role={role} />
      <PropertyGallery photos={property.photos} propertyName={property.name} />
      <div className="flex flex-col gap-4 px-4 py-4">
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <GenderBadge gender={property.gender} />
            {property.is_flash_sale && <Badge color="brand">Promo</Badge>}
          </div>
          <h1 className="text-title-lg font-bold text-neutral-900">{property.name}</h1>
          <p className="text-caption text-neutral-600">
            {property.address}, {property.area.name}{" "}
            <a
              href={mapsUrl}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-brand-700"
            >
              Buka di peta
              <ExternalLink aria-hidden="true" size={14} strokeWidth={1.75} />
            </a>
          </p>
          <RatingSummary average={property.rating.average} count={property.rating.count} />
        </div>
      </div>
      <KostDetailClient propertyId={property.id} description={property.description} rooms={rooms} />
    </>
  );
}
