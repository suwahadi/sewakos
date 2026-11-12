import { Building2 } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { GenderBadge } from "@/components/ui/gender-badge";
import { RatingStars } from "@/components/ui/rating";
import { Skeleton } from "@/components/ui/skeleton";
import { formatRating } from "@/lib/format";
import { cn } from "@/lib/cn";

export interface ListingCardRoom {
  id: number;
  property_id: number;
  name: string;
  property_name: string;
  area_name: string;
  thumbnail_url: string | null;
  gender: string;
  price_label: string;
  available: number;
  is_flash_sale: boolean;
  rating_average: number | null;
  rating_count: number;
}

const RENT_TYPE_SUFFIX: Record<1 | 2 | 3, string> = { 1: "/hari", 2: "/minggu", 3: "/bulan" };

export function ListingCard({
  room,
  rentType = 3,
  vertical = false,
  className,
}: {
  room: ListingCardRoom;
  rentType?: 1 | 2 | 3;
  vertical?: boolean;
  className?: string;
}) {
  const availabilityLabel =
    room.available === 0 ? "Penuh" : room.available <= 3 ? `Sisa ${room.available} kamar` : null;

  return (
    <Link
      href={`/kost/${room.property_id}`}
      className={cn(
        "flex h-full min-w-0",
        vertical ? "flex-col gap-2" : "gap-3 sm:flex-col sm:gap-2",
        className,
      )}
    >
      <div
        className={cn(
          "relative shrink-0 overflow-hidden rounded-md bg-neutral-100",
          vertical ? "aspect-[4/3] w-full" : "aspect-[4/5] w-28 sm:aspect-[4/3] sm:w-full",
        )}
      >
        {room.thumbnail_url ? (
          <Image
            src={room.thumbnail_url}
            alt=""
            fill
            sizes={vertical ? "192px" : "(min-width:1024px) 25vw, (min-width:640px) 50vw, 112px"}
            className="object-cover"
          />
        ) : (
          <span className="flex size-full items-center justify-center">
            <Building2 aria-hidden="true" size={32} strokeWidth={1.5} className="text-neutral-400" />
          </span>
        )}
        {room.is_flash_sale && (
          <Badge color="brand" className="absolute left-2 top-2">
            Promo
          </Badge>
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center gap-1.5">
          <GenderBadge gender={room.gender} />
          <span className="truncate text-caption text-neutral-600">{room.area_name}</span>
        </div>
        <p className="truncate text-title-sm font-semibold text-neutral-900">{room.property_name}</p>
        <p className="truncate text-caption text-neutral-600">{room.name}</p>
        <div className="flex h-[18px] items-center gap-3 text-caption text-neutral-600">
          {room.rating_count > 0 && (
            <span className="inline-flex shrink-0 items-center gap-1">
              <RatingStars size={14} />
              {formatRating(room.rating_average ?? 0)} ({room.rating_count})
            </span>
          )}
          {availabilityLabel && <span className="truncate">{availabilityLabel}</span>}
        </div>
        <p className="mt-auto text-price font-bold text-neutral-900">
          {room.price_label}
          <span className="text-body font-normal text-neutral-600"> {RENT_TYPE_SUFFIX[rentType]}</span>
        </p>
      </div>
    </Link>
  );
}

export function ListingCardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("flex h-full min-w-0 gap-3 sm:flex-col sm:gap-0", className)}>
      <Skeleton className="aspect-[4/5] w-28 shrink-0 rounded-md sm:mb-2 sm:aspect-[4/3] sm:w-full" />
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <Skeleton className="h-[18px] w-24 rounded-sm" />
        <Skeleton className="h-[18px] w-3/4 rounded-sm" />
        <Skeleton className="h-[18px] w-1/2 rounded-sm" />
        <Skeleton className="mt-auto h-[22px] w-28 rounded-sm" />
      </div>
    </div>
  );
}
