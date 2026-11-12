"use client";

import { useState, type ReactNode } from "react";
import { Avatar } from "@/components/ui/avatar";
import { RatingStarsRow } from "@/components/ui/rating";
import { cn } from "@/lib/cn";
import { formatRelativeTime } from "@/lib/format";
import type { Review } from "@/lib/reviews";

const LONG_COMMENT = 200;

export function ReviewCard({ review, children }: { review: Review; children?: ReactNode }) {
  const [expanded, setExpanded] = useState(false);
  const long = (review.comment?.length ?? 0) > LONG_COMMENT;

  return (
    <article className="flex gap-3 py-4">
      <Avatar name={review.tenant_name} />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div>
          <p className="text-body font-semibold text-neutral-900">{review.tenant_name}</p>
          <p className="text-caption text-neutral-600">{review.room_name}</p>
        </div>
        <RatingStarsRow rating={review.rating} />
        {review.comment && (
          <p className={cn("whitespace-pre-line break-words text-body text-neutral-800", long && !expanded && "line-clamp-4")}>
            {review.comment}
          </p>
        )}
        {long && (
          <button
            type="button"
            onClick={() => setExpanded((value) => !value)}
            className="self-start text-label font-semibold text-brand-700"
            aria-expanded={expanded}
          >
            {expanded ? "Tutup" : "Selengkapnya"}
          </button>
        )}
        <p className="text-caption text-neutral-600">{formatRelativeTime(review.created_at)}</p>
        {children}
      </div>
    </article>
  );
}
