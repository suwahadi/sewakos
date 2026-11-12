import { AdminReviewList } from "@/components/features/admin-review-list";
import { ErrorState } from "@/components/ui/error-state";
import { ADMIN_PAGE_SIZE } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";
import type { AdminReview } from "@/lib/reviews";

export const metadata = { title: "Moderasi ulasan" };

export default async function AdminReviewsPage() {
  const result = await apiFetch<{ reviews: AdminReview[] }>("/admin/reviews", {
    searchParams: { is_hidden: "false", limit: ADMIN_PAGE_SIZE },
  });
  return result.ok ? <AdminReviewList initial={result.data.reviews} /> : <ErrorState title={result.meta.message} />;
}
