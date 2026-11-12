export const REVIEW_PAGE_SIZE = 10;

export interface Review {
  id: number;
  rating: number;
  comment: string | null;
  tenant_name: string;
  room_name: string;
  created_at: string;
}

export interface Rating {
  average: number | null;
  count: number;
}

export interface ReviewsResponse {
  rating: Rating;
  reviews: Review[];
  current_page: number;
}

export interface OwnerReview extends Review {
  property_id: number;
  is_hidden: boolean;
}

export interface AdminReview extends OwnerReview {
  hidden_reason: string | null;
}
