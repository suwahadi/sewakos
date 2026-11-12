import { apiFetch } from "@/lib/api/client";
import { REVIEW_PAGE_SIZE, type Review, type ReviewsResponse } from "@/lib/reviews";

export interface PropertyReviewsData {
  property: { id: number; name: string };
  rating: ReviewsResponse["rating"];
  reviews: Review[];
}

export async function loadPropertyReviews(id: string): Promise<PropertyReviewsData | null> {
  const [listing, result] = await Promise.all([
    apiFetch<{ property: { id: number; name: string } }>(`/listings/${encodeURIComponent(id)}`),
    apiFetch<ReviewsResponse>(`/listings/${encodeURIComponent(id)}/reviews`, {
      auth: false,
      searchParams: { limit: REVIEW_PAGE_SIZE },
    }),
  ]);
  if (!listing.ok || !result.ok) return null;
  return { property: listing.data.property, rating: result.data.rating, reviews: result.data.reviews };
}
