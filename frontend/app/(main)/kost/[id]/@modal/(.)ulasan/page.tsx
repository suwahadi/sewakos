import { ReviewsPopup } from "@/components/features/reviews-popup";
import { loadPropertyReviews } from "@/lib/listing-reviews";

export default async function KostReviewsPopupPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const data = await loadPropertyReviews(id);
  if (!data) return null;
  return (
    <ReviewsPopup
      propertyId={data.property.id}
      propertyName={data.property.name}
      rating={data.rating}
      reviews={data.reviews}
    />
  );
}
