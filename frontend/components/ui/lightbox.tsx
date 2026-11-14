"use client";

import { ChevronLeft, ChevronRight, X } from "lucide-react";
import Image from "next/image";
import { useEffect, useRef, type KeyboardEvent } from "react";
import { closeDialog, openDialog } from "@/lib/dialog";
import { IconButton } from "./icon-button";

export interface LightboxPhoto {
  src: string;
  alt: string;
}

const SWIPE_THRESHOLD_PX = 50;
const ON_DARK = "text-neutral-0 hover:bg-neutral-0/15";

export function Lightbox({
  photos,
  index,
  onIndexChange,
  onClose,
  label,
}: {
  photos: LightboxPhoto[];
  index: number | null;
  onIndexChange: (index: number) => void;
  onClose: () => void;
  label: string;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const touchStartX = useRef<number | null>(null);
  const photo = index === null ? undefined : photos[index];

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (index !== null && !dialog.open) openDialog(dialog);
    if (index === null && dialog.open) closeDialog(dialog);
  }, [index]);

  function step(delta: number) {
    if (index === null) return;
    onIndexChange((index + delta + photos.length) % photos.length);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLDialogElement>) {
    if (event.key === "ArrowLeft") step(-1);
    if (event.key === "ArrowRight") step(1);
  }

  return (
    <dialog
      ref={dialogRef}
      aria-label={label}
      className="m-0 h-dvh max-h-none w-screen max-w-none border-0 bg-neutral-900 p-0 backdrop:bg-neutral-900"
      onKeyDown={handleKeyDown}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <div
        className="relative flex size-full items-center justify-center"
        onClick={(event) => {
          if (event.target === event.currentTarget) onClose();
        }}
        onTouchStart={(event) => {
          touchStartX.current = event.touches[0]?.clientX ?? null;
        }}
        onTouchEnd={(event) => {
          const start = touchStartX.current;
          const end = event.changedTouches[0]?.clientX;
          touchStartX.current = null;
          if (start === null || end === undefined || Math.abs(end - start) < SWIPE_THRESHOLD_PX) return;
          step(end < start ? 1 : -1);
        }}
      >
        {photo && (
          <div className="relative h-[85dvh] w-full max-w-[1200px]">
            <Image src={photo.src} alt={photo.alt} fill sizes="100vw" className="object-contain" />
          </div>
        )}
        <IconButton
          icon={<X aria-hidden="true" />}
          aria-label="Tutup"
          onClick={onClose}
          className={`absolute right-3 top-3 ${ON_DARK}`}
        />
        {photos.length > 1 && (
          <>
            <IconButton
              icon={<ChevronLeft aria-hidden="true" />}
              aria-label="Foto sebelumnya"
              onClick={() => step(-1)}
              className={`absolute left-3 top-1/2 -translate-y-1/2 ${ON_DARK}`}
            />
            <IconButton
              icon={<ChevronRight aria-hidden="true" />}
              aria-label="Foto berikutnya"
              onClick={() => step(1)}
              className={`absolute right-3 top-1/2 -translate-y-1/2 ${ON_DARK}`}
            />
          </>
        )}
      </div>
    </dialog>
  );
}
