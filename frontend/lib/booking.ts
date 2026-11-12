import type { RentType } from "./period";

export const BOOKING_PAGE_SIZE = 20;

export interface BookingSummary {
  id: number;
  status: string;
  rent_type: RentType;
  room: { id: number; name: string };
  start_date: string;
  duration: number;
  end_date: string;
  price_label: string;
  expires_at: string;
}

export interface BookingDetail extends BookingSummary {
  property: { id: number; name: string };
  deposit_label: string;
  admin_fee: number;
  admin_fee_label: string;
  total_payment_label: string;
  paid_at: string | null;
  reject_reason: string | null;
  can_extend: boolean;
  can_review: boolean;
}

export interface Payment {
  status: string;
  expires_at: string;
}

export interface Refund {
  amount_label: string;
  status: "awaiting_account" | "ready" | "completed";
  completed_at: string | null;
}

export interface Invoice {
  id: number;
  number: string;
  type: string;
  status: string;
  period_start: string;
  period_end: string;
  total_label: string;
  due_date: string;
}
