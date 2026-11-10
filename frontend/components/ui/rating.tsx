import { Star } from "lucide-react";
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
