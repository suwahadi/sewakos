"use client";

import { MessageSquareText } from "lucide-react";
import { useRouter } from "next/navigation";
import { useRef } from "react";
import { ReviewList } from "@/components/features/review-list";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { EmptyState } from "@/components/ui/empty-state";
import { RatingSummary } from "@/components/ui/rating";
import type { Rating, Review } from "@/lib/reviews";

export interface ReviewsPopupProps {
  propertyId: number;
  propertyName: string;
  rating: Rating;
  reviews: Review[];
}

export function ReviewsPopup({ propertyId, propertyName, rating, reviews }: ReviewsPopupProps) {
  const router = useRouter();
  const closed = useRef(false);

  function close() {
    if (closed.current) return;
    closed.current = true;
    router.back();
  }

  return (
    <BottomSheet open title={`Ulasan ${propertyName}`} onClose={close}>
      <div className="flex flex-col gap-4 pb-6">
        <RatingSummary average={rating.average} count={rating.count} />
        {reviews.length > 0 ? (
          <ReviewList propertyId={propertyId} initial={reviews} />
        ) : (
          <EmptyState icon={MessageSquareText} title="Belum ada ulasan" />
        )}
      </div>
    </BottomSheet>
  );
}
