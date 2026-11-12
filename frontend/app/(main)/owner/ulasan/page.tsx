import { AppHeader } from "@/components/features/app-header";
import { OwnerReviewList } from "@/components/features/owner-review-list";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import type { OwnerProperty } from "@/lib/owner";
import { REVIEW_PAGE_SIZE, type OwnerReview } from "@/lib/reviews";

export const metadata = { title: "Ulasan properti" };

export default async function OwnerReviewsPage() {
  await requireAuthenticated("/owner/ulasan");
  const [role, properties, reviews] = await Promise.all([
    getNavRole(),
    apiFetch<{ properties: OwnerProperty[] }>("/owner/properties", { searchParams: { limit: 50 } }),
    apiFetch<{ reviews: OwnerReview[] }>("/owner/reviews", { searchParams: { limit: REVIEW_PAGE_SIZE } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Ulasan properti" role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : !reviews.ok ? (
        <ErrorState title={reviews.meta.message} />
      ) : (
        <main className="mx-auto max-w-[720px] px-4 py-4">
          <OwnerReviewList
            properties={properties.ok ? properties.data.properties.map(({ id, name }) => ({ id, name })) : []}
            initial={reviews.data.reviews}
          />
        </main>
      )}
    </>
  );
}
