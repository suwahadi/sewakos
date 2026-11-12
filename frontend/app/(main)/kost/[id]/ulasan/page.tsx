import { MessageSquareText } from "lucide-react";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { ReviewList } from "@/components/features/review-list";
import { EmptyState } from "@/components/ui/empty-state";
import { RatingSummary } from "@/components/ui/rating";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";
import { REVIEW_PAGE_SIZE, type ReviewsResponse } from "@/lib/reviews";

export const metadata: Metadata = { title: "Ulasan" };

export default async function KostReviewsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [role, listing, result] = await Promise.all([
    getNavRole(),
    apiFetch<{ property: { id: number; name: string } }>(`/listings/${encodeURIComponent(id)}`),
    apiFetch<ReviewsResponse>(`/listings/${encodeURIComponent(id)}/reviews`, { auth: false, searchParams: { limit: REVIEW_PAGE_SIZE } }),
  ]);
  if (!listing.ok || !result.ok) notFound();
  const { property } = listing.data;
  const { rating, reviews } = result.data;

  return (
    <>
      <AppHeader variant="inner" title={`Ulasan ${property.name}`} role={role} />
      <main className="flex flex-col gap-4 px-4 py-4">
        <RatingSummary average={rating.average} count={rating.count} />
        {reviews.length > 0 ? (
          <ReviewList propertyId={property.id} initial={reviews} />
        ) : (
          <EmptyState icon={MessageSquareText} title="Belum ada ulasan" />
        )}
      </main>
    </>
  );
}
