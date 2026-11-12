import { MessageSquareText } from "lucide-react";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { ReviewList } from "@/components/features/review-list";
import { EmptyState } from "@/components/ui/empty-state";
import { RatingSummary } from "@/components/ui/rating";
import { getNavRole } from "@/lib/auth/nav-role";
import { loadPropertyReviews } from "@/lib/listing-reviews";

export const metadata: Metadata = { title: "Ulasan" };

export default async function KostReviewsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [role, data] = await Promise.all([getNavRole(), loadPropertyReviews(id)]);
  if (!data) notFound();

  return (
    <>
      <AppHeader variant="inner" title={`Ulasan ${data.property.name}`} role={role} />
      <main className="flex flex-col gap-4 px-4 py-4">
        <RatingSummary average={data.rating.average} count={data.rating.count} />
        {data.reviews.length > 0 ? (
          <ReviewList propertyId={data.property.id} initial={data.reviews} />
        ) : (
          <EmptyState icon={MessageSquareText} title="Belum ada ulasan" />
        )}
      </main>
    </>
  );
}
