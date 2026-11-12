"use client";

import { Star } from "lucide-react";
import { useId, useRef, type KeyboardEvent } from "react";
import { RATING_LABELS } from "@/components/ui/rating";
import { cn } from "@/lib/cn";

const STARS = [1, 2, 3, 4, 5];

export function RatingInput({ value, onChange, error }: { value: number; onChange: (rating: number) => void; error?: string }) {
  const buttons = useRef<(HTMLButtonElement | null)[]>([]);
  const id = useId();

  function select(rating: number) {
    onChange(rating);
    buttons.current[rating - 1]?.focus();
  }

  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    const current = value || 1;
    if (event.key === "ArrowRight" || event.key === "ArrowUp") {
      event.preventDefault();
      select(Math.min(5, value ? current + 1 : 1));
    } else if (event.key === "ArrowLeft" || event.key === "ArrowDown") {
      event.preventDefault();
      select(Math.max(1, current - 1));
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <p id={`${id}-label`} className="text-label font-medium text-neutral-700">
        Rating
      </p>
      <div
        role="radiogroup"
        aria-labelledby={`${id}-label`}
        aria-describedby={error ? `${id}-error` : undefined}
        className="flex gap-1"
        onKeyDown={onKeyDown}
      >
        {STARS.map((star) => (
          <button
            key={star}
            ref={(element) => {
              buttons.current[star - 1] = element;
            }}
            type="button"
            role="radio"
            aria-checked={value === star}
            aria-label={`${star} bintang`}
            tabIndex={value === star || (value === 0 && star === 1) ? 0 : -1}
            onClick={() => select(star)}
            className="inline-flex size-10 items-center justify-center rounded-md focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-brand-100"
          >
            <Star
              aria-hidden="true"
              size={28}
              strokeWidth={1.75}
              fill={star <= value ? "currentColor" : "none"}
              className={cn(star <= value ? "text-star" : "text-neutral-400")}
            />
          </button>
        ))}
      </div>
      <p className="text-caption text-neutral-600">{value ? RATING_LABELS[value - 1] : "Pilih 1 sampai 5 bintang"}</p>
      {error && (
        <p id={`${id}-error`} className="text-caption text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
