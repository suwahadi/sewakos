"use client";

import { Camera } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import { Lightbox } from "@/components/ui/lightbox";

export interface GalleryPhoto {
  url: string;
  thumbnail_url: string;
  caption: string | null;
}

export function PropertyGallery({ photos, propertyName }: { photos: GalleryPhoto[]; propertyName: string }) {
  const [index, setIndex] = useState(0);
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);

  if (photos.length === 0) {
    return (
      <div className="px-4 pt-4 lg:pt-6">
        <div className="flex aspect-[4/3] items-center justify-center rounded-md bg-neutral-100 lg:aspect-auto lg:h-80">
          <span className="flex flex-col items-center gap-2 text-neutral-400">
            <Camera aria-hidden="true" size={40} strokeWidth={1.5} />
            <span className="text-body text-neutral-600">Belum ada foto</span>
          </span>
        </div>
      </div>
    );
  }

  function altFor(photo: GalleryPhoto, position: number) {
    return photo.caption ?? `Foto ${propertyName} ${position}`;
  }

  return (
    <div className="px-4 pt-4 lg:pt-6">
      <div className="relative overflow-hidden rounded-md lg:hidden">
        <div
          className="flex snap-x snap-mandatory overflow-x-auto [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
          onScroll={(event) => {
            const target = event.currentTarget;
            setIndex(Math.round(target.scrollLeft / target.clientWidth));
          }}
        >
          {photos.map((photo, i) => (
            <button
              key={photo.url}
              type="button"
              aria-label={`Perbesar foto ${i + 1}`}
              onClick={() => setLightboxIndex(i)}
              className="relative aspect-[4/3] w-full shrink-0 snap-start"
            >
              <Image
                src={photo.url}
                alt={altFor(photo, i + 1)}
                fill
                sizes="100vw"
                priority={i === 0}
                loading={i === 0 ? undefined : "lazy"}
                className="object-cover"
              />
            </button>
          ))}
        </div>
        <p className="pointer-events-none absolute bottom-2 right-2 rounded-sm bg-neutral-900/64 px-2 py-1 text-caption text-neutral-0">
          {index + 1}/{photos.length}
        </p>
      </div>

      <div className="hidden gap-2 lg:grid lg:grid-cols-3">
        <button
          type="button"
          aria-label="Perbesar foto 1"
          onClick={() => setLightboxIndex(0)}
          className="relative col-span-2 aspect-[16/9]"
        >
          <Image src={photos[0]!.url} alt={altFor(photos[0]!, 1)} fill sizes="66vw" priority className="rounded-md object-cover" />
        </button>
        <div className="grid grid-rows-2 gap-2">
          {photos.slice(1, 3).map((photo, i) => (
            <button
              key={photo.url}
              type="button"
              aria-label={`Perbesar foto ${i + 2}`}
              onClick={() => setLightboxIndex(i + 1)}
              className="relative aspect-[16/9]"
            >
              <Image src={photo.url} alt={altFor(photo, i + 2)} fill sizes="33vw" loading="lazy" className="rounded-md object-cover" />
            </button>
          ))}
        </div>
      </div>
      {photos.length > 1 && (
        <button
          type="button"
          onClick={() => setLightboxIndex(0)}
          className="mt-2 hidden text-label font-semibold text-brand-700 lg:block"
        >
          Lihat semua foto
        </button>
      )}

      <Lightbox
        photos={photos.map((photo, i) => ({ src: photo.url, alt: altFor(photo, i + 1) }))}
        index={lightboxIndex}
        onIndexChange={setLightboxIndex}
        onClose={() => setLightboxIndex(null)}
        label={`Foto ${propertyName}`}
      />
    </div>
  );
}
