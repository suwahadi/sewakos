"use client";

import { MessageSquareText } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ReviewCard } from "@/components/features/review-card";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { LoadMore } from "@/components/ui/load-more";
import { Select } from "@/components/ui/select";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { REVIEW_PAGE_SIZE, type OwnerReview } from "@/lib/reviews";

export function OwnerReviewList({ properties, initial }: { properties: { id: number; name: string }[]; initial: OwnerReview[] }) {
  const router = useRouter();
  const [reviews, setReviews] = useState(initial);
  const [propertyId, setPropertyId] = useState("");
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= REVIEW_PAGE_SIZE);
  const [loading, setLoading] = useState(false);
  const names = new Map(properties.map((property) => [property.id, property.name]));

  async function load(nextPropertyId: string, nextPage: number) {
    setLoading(true);
    const query = new URLSearchParams({ page: String(nextPage), limit: String(REVIEW_PAGE_SIZE) });
    if (nextPropertyId) query.set("property_id", nextPropertyId);
    const result = await callBff<{ reviews: OwnerReview[] }>(`/api/owner/reviews?${query}`);
    setLoading(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), () => load(nextPropertyId, nextPage));
      return;
    }
    setReviews((current) => (nextPage === 1 ? result.data.reviews : [...current, ...result.data.reviews]));
    setPage(nextPage);
    setHasMore(result.data.reviews.length >= REVIEW_PAGE_SIZE);
  }

  return (
    <div className="flex flex-col gap-4">
      <Select
        label="Properti"
        value={propertyId}
        onChange={(event) => {
          setPropertyId(event.target.value);
          load(event.target.value, 1);
        }}
      >
        <option value="">Semua properti</option>
        {properties.map((property) => (
          <option key={property.id} value={property.id}>
            {property.name}
          </option>
        ))}
      </Select>
      {reviews.length === 0 && !loading ? (
        <EmptyState icon={MessageSquareText} title="Belum ada ulasan" />
      ) : (
        <div className="divide-y divide-neutral-200">
          {reviews.map((review) => (
            <ReviewCard key={review.id} review={review}>
              <p className="text-caption text-neutral-600">{names.get(review.property_id) ?? `Properti #${review.property_id}`}</p>
              {review.is_hidden && (
                <Badge color="neutral" className="self-start">
                  Disembunyikan admin
                </Badge>
              )}
            </ReviewCard>
          ))}
        </div>
      )}
      {hasMore && <LoadMore loading={loading} onClick={() => load(propertyId, page + 1)} />}
    </div>
  );
}
