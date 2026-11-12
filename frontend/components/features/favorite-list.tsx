"use client";

import { Building2, Heart } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { FavoriteButton } from "@/components/features/favorite-button";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { GenderBadge } from "@/components/ui/gender-badge";
import { LoadMore } from "@/components/ui/load-more";
import { RatingStars } from "@/components/ui/rating";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatRating } from "@/lib/format";

export const FAVORITE_PAGE_SIZE = 20;

export interface Favorite {
  id: number;
  name: string;
  gender: string;
  area: { name: string };
  thumbnail_url: string | null;
  min_price_monthly_label: string;
  rating_average: number | null;
  is_available: boolean;
}

export function FavoriteList({ initial }: { initial: Favorite[] }) {
  const router = useRouter();
  const [favorites, setFavorites] = useState(initial);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= FAVORITE_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    setLoading(true);
    const next = page + 1;
    const result = await callBff<{ favorites: Favorite[] }>(`/api/favorites?page=${next}&limit=${FAVORITE_PAGE_SIZE}`);
    setLoading(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), loadMore);
      return;
    }
    setFavorites((current) => [...current, ...result.data.favorites]);
    setPage(next);
    setHasMore(result.data.favorites.length >= FAVORITE_PAGE_SIZE);
  }

  if (favorites.length === 0) {
    return (
      <EmptyState
        icon={Heart}
        title="Belum ada kost favorit"
        description="Simpan kost yang kamu suka agar mudah dibuka lagi."
        action={{ label: "Cari kost", onClick: () => router.push("/cari") }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <ul className="flex flex-col divide-y divide-neutral-200">
        {favorites.map((favorite) => (
          <li key={favorite.id} className="flex items-start gap-3 py-3">
            <FavoriteCard favorite={favorite} />
            <FavoriteButton
              propertyId={favorite.id}
              initial
              guest={false}
              variant="plain"
              onRemoved={() => setFavorites((current) => current.filter((item) => item.id !== favorite.id))}
            />
          </li>
        ))}
      </ul>
      {hasMore && <LoadMore loading={loading} onClick={loadMore} />}
    </div>
  );
}

function FavoriteCard({ favorite }: { favorite: Favorite }) {
  const body = (
    <>
      <div className="relative aspect-[4/3] w-28 shrink-0 overflow-hidden rounded-md bg-neutral-100">
        {favorite.thumbnail_url ? (
          <Image src={favorite.thumbnail_url} alt="" fill sizes="112px" className="object-cover" />
        ) : (
          <span className="flex size-full items-center justify-center">
            <Building2 aria-hidden="true" size={32} strokeWidth={1.5} className="text-neutral-400" />
          </span>
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center gap-1.5">
          <GenderBadge gender={favorite.gender} />
          <span className="truncate text-caption text-neutral-600">{favorite.area.name}</span>
        </div>
        <p className="truncate text-title-sm font-semibold text-neutral-900">{favorite.name}</p>
        {favorite.is_available ? (
          <p className="text-body text-neutral-900">
            <span className="font-semibold">{favorite.min_price_monthly_label}</span>
            <span className="text-caption text-neutral-600">/bulan</span>
          </p>
        ) : (
          <Badge color="neutral" className="self-start">
            Tidak tersedia
          </Badge>
        )}
        {favorite.rating_average !== null && (
          <span className="inline-flex items-center gap-1 text-caption text-neutral-600">
            <RatingStars size={14} />
            {formatRating(favorite.rating_average)}
          </span>
        )}
      </div>
    </>
  );

  return favorite.is_available ? (
    <Link href={`/kost/${favorite.id}`} className="flex min-w-0 flex-1 gap-3">
      {body}
    </Link>
  ) : (
    <div className="flex min-w-0 flex-1 gap-3 opacity-70">{body}</div>
  );
}
