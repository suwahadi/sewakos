import { ChevronRight, ExternalLink } from "lucide-react";
import Link from "next/link";
import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { FavoriteButton } from "@/components/features/favorite-button";
import { KostDetailClient } from "@/components/features/kost-detail-client";
import { PropertyGallery, type GalleryPhoto } from "@/components/features/property-gallery";
import { ListingGrid } from "@/components/features/listing-grid";
import type { ListingCardRoom } from "@/components/features/listing-card";
import { ReviewCard } from "@/components/features/review-card";
import type { RoomOptionData } from "@/components/features/room-option";
import { Badge } from "@/components/ui/badge";
import { GenderBadge } from "@/components/ui/gender-badge";
import { RatingInline, RatingSummary } from "@/components/ui/rating";
import { apiFetch } from "@/lib/api/client";
import { pickRandom } from "@/lib/random";
import { getNavRole } from "@/lib/auth/nav-role";
import type { ReviewsResponse } from "@/lib/reviews";

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
    is_favorite: boolean;
    area: { name: string };
    photos: GalleryPhoto[];
    rating: { average: number | null; count: number };
  };
  rooms: RoomOptionData[];
}

async function getOtherRandomRooms(propertyId: number): Promise<ListingCardRoom[]> {
  const result = await apiFetch<{ rooms: ListingCardRoom[] }>("/listings/latest", { auth: false, searchParams: { limit: "50" } });
  if (!result.ok) return [];
  const seen = new Set<number>([propertyId]);
  const oneRoomPerProperty = result.data.rooms.filter((room) => {
    if (seen.has(room.property_id)) return false;
    seen.add(room.property_id);
    return true;
  });
  return pickRandom(oneRoomPerProperty, 6);
}

export default async function KostDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [role, result, reviewsResult] = await Promise.all([
    getNavRole(),
    apiFetch<PropertyDetailResponse>(`/listings/${encodeURIComponent(id)}`),
    apiFetch<ReviewsResponse>(`/listings/${encodeURIComponent(id)}/reviews`, { auth: false, searchParams: { limit: 3 } }),
  ]);

  if (!result.ok) notFound();
  const { property, rooms } = result.data;
  const otherRooms = await getOtherRandomRooms(property.id);
  const reviews = reviewsResult.ok ? reviewsResult.data.reviews : [];
  const mapsUrl = `https://www.google.com/maps/search/?api=1&query=${property.latitude},${property.longitude}`;

  return (
    <>
      <AppHeader
        variant="inner"
        title={property.name}
        role={role}
        rightAction={
          role !== "owner" && (
            <FavoriteButton propertyId={property.id} initial={property.is_favorite} guest={role === "guest"} variant="onBrand" />
          )
        }
      />
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
          <RatingInline average={property.rating.average} count={property.rating.count} />
        </div>
      </div>
      <KostDetailClient
        propertyId={property.id}
        similar={
          otherRooms.length > 0 && (
            <section className="flex flex-col gap-3">
              <p className="text-title-sm font-semibold text-neutral-900">Kost lainnya</p>
              <ListingGrid rooms={otherRooms} layout="detail" />
            </section>
          )
        }
        description={property.description}
        rooms={rooms}
        reviews={
          <section className="flex flex-col gap-3">
            <p className="text-title-sm font-semibold text-neutral-900">Ulasan</p>
            <RatingSummary average={property.rating.average} count={property.rating.count} />
            {reviews.length > 0 && (
              <div className="divide-y divide-neutral-200">
                {reviews.map((review) => (
                  <ReviewCard key={review.id} review={review} />
                ))}
              </div>
            )}
            {property.rating.count > reviews.length && (
              <Link
                href={`/kost/${property.id}/ulasan`}
                className="inline-flex min-h-11 items-center gap-1 text-label font-semibold text-brand-700"
              >
                Lihat semua ulasan ({property.rating.count})
                <ChevronRight aria-hidden="true" size={16} strokeWidth={1.75} />
              </Link>
            )}
          </section>
        }
      />
    </>
  );
}
