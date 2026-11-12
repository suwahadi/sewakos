"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ReviewCard } from "@/components/features/review-card";
import { LoadMore } from "@/components/ui/load-more";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { REVIEW_PAGE_SIZE, type Review, type ReviewsResponse } from "@/lib/reviews";

export function ReviewList({ propertyId, initial }: { propertyId: number; initial: Review[] }) {
  const router = useRouter();
  const [reviews, setReviews] = useState(initial);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= REVIEW_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    setLoading(true);
    const next = page + 1;
    const result = await callBff<ReviewsResponse>(`/api/listings/${propertyId}/reviews?page=${next}&limit=${REVIEW_PAGE_SIZE}`);
    setLoading(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), loadMore);
      return;
    }
    setReviews((current) => [...current, ...result.data.reviews]);
    setPage(next);
    setHasMore(result.data.reviews.length >= REVIEW_PAGE_SIZE);
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="divide-y divide-neutral-200">
        {reviews.map((review) => (
          <ReviewCard key={review.id} review={review} />
        ))}
      </div>
      {hasMore && <LoadMore loading={loading} onClick={loadMore} />}
    </div>
  );
}
