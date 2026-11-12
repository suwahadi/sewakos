import Image from "next/image";
import Link from "next/link";
import type { AreaChipOption } from "./area-chip-row";

export interface AreaBannerOption extends AreaChipOption {
  slug: string;
}

// Photos: Unsplash License (https://unsplash.com/license), served from images.unsplash.com.
// shortcut: photos are keyed by area slug in code because areas have no image column; a city added
// later shows the brand-colour card until its photo is added here.
const AREA_PHOTOS: Record<string, string> = {
  "jakarta-selatan": "photo-1567756396406-f529ebb5583d",
  "jakarta-pusat": "photo-1531453213298-0006c70a1671",
  "kota-depok": "photo-1688361496231-7194bdf2f592",
  "kota-bandung": "photo-1707993467310-a5b2bb858d68",
  "kota-yogyakarta": "photo-1631795617958-3ddcf718d6aa",
  "kota-surabaya": "photo-1566176553949-872b2a73e04e",
  "kota-malang": "photo-1725360255585-20dbd9803917",
  "kota-semarang": "photo-1651890053473-b25f7e1672dd",
};

export function areaPhotoUrl(slug: string): string | null {
  const photo = AREA_PHOTOS[slug];
  return photo ? `https://images.unsplash.com/${photo}?auto=format&fit=crop&w=960&q=75` : null;
}

export function AreaBannerGrid({ areas }: { areas: AreaBannerOption[] }) {
  if (areas.length === 0) return null;

  return (
    <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:gap-4">
      {areas.map((area) => {
        const photo = areaPhotoUrl(area.slug);
        return (
          <li key={area.id}>
            <Link
              href={`/cari?area_id=${area.id}&area_name=${encodeURIComponent(area.name)}`}
              className="group relative flex aspect-[4/3] items-center justify-center overflow-hidden rounded-md bg-brand-600 lg:aspect-[16/9]"
            >
              {photo && (
                <>
                  <Image
                    src={photo}
                    alt=""
                    fill
                    sizes="(min-width:640px) 33vw, 50vw"
                    className="object-cover transition-transform duration-[var(--duration-slow)] ease-standard motion-safe:group-hover:scale-105"
                  />
                  <span aria-hidden="true" className="absolute inset-0 bg-neutral-900/60" />
                </>
              )}
              <span className="relative px-3 text-center text-title-md font-bold text-neutral-0 sm:text-title-lg">
                {area.name}
              </span>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
