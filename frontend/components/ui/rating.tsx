import { Star } from "lucide-react";
import { cn } from "@/lib/cn";
import { formatRating } from "@/lib/format";

export function RatingStars({ size = 16 }: { size?: number }) {
  return <Star aria-hidden="true" size={size} strokeWidth={1.75} fill="currentColor" className="text-star" />;
}

export interface RatingSummaryProps {
  average: number | null;
  count: number;
}

export function RatingSummary({ average, count }: RatingSummaryProps) {
  if (average === null) {
    return <p className="text-body text-neutral-600">Belum ada ulasan</p>;
  }

  return (
    <div
      className="flex items-center gap-2"
      aria-label={`Rating ${formatRating(average)} dari 5, ${count} ulasan`}
    >
      <span className="text-title-lg font-bold text-neutral-900">{formatRating(average)}</span>
      <RatingStars size={20} />
      <span className="text-caption text-neutral-600">({count} ulasan)</span>
    </div>
  );
}

export function RatingInline({ average, count }: RatingSummaryProps) {
  if (average === null) {
    return <p className="text-caption text-neutral-600">Belum ada ulasan</p>;
  }

  return (
    <p className="flex items-center gap-1" aria-label={`Rating ${formatRating(average)} dari 5, ${count} ulasan`}>
      <RatingStars />
      <span className="text-body font-semibold text-neutral-900">{formatRating(average)}</span>
      <span className="text-caption text-neutral-600">({count} ulasan)</span>
    </p>
  );
}

export function RatingStarsRow({ rating }: { rating: number }) {
  return (
    <span role="img" aria-label={`Rating ${rating} dari 5`} className="flex items-center gap-0.5">
      {[1, 2, 3, 4, 5].map((star) => (
        <Star
          key={star}
          aria-hidden="true"
          size={14}
          strokeWidth={1.75}
          fill={star <= rating ? "currentColor" : "none"}
          className={cn(star <= rating ? "text-star" : "text-neutral-300")}
        />
      ))}
    </span>
  );
}

export const RATING_LABELS = ["Buruk", "Kurang", "Cukup", "Bagus", "Sangat bagus"] as const;
