export const OWNER_PAGE_SIZE = 20;

export interface OwnerBooking {
  id: number;
  status: string;
  rent_type: string;
  start_date: string;
  duration: number;
  end_date: string;
  tenant: { id: number; name: string; phone_number: string | null };
  property: { id: number; name: string };
  room: { id: number; name: string };
  price_label: string;
  deposit_label: string;
  payout_amount_label: string;
  paid_at: string | null;
  owner_deadline_at: string | null;
  reject_reason: string | null;
  deposit_returned_at: string | null;
}

export interface OwnerPhoto {
  id: number;
  url: string;
  thumbnail_url: string;
  caption: string | null;
  room_id: number | null;
  sort_order: number;
}

export interface OwnerProperty {
  id: number;
  name: string;
  gender: string;
  description: string;
  address: string;
  latitude: number;
  longitude: number;
  area: { id: number; name: string };
  is_active: boolean;
  moderation_status: string;
  moderation_note: string | null;
  is_flash_sale: boolean;
  flash_sale_discount: number;
  rooms_count: number;
  photos: OwnerPhoto[];
}

export interface OwnerRoom {
  id: number;
  property_id: number;
  name: string;
  price_monthly: number;
  price_weekly: number | null;
  price_daily: number | null;
  available: number;
  held: number;
  occupied: number;
  capacity: number;
  monthly_duration_options: number[];
  facilities: string[];
}

export interface BankAccount {
  bank_name: string;
  account_number: string;
  account_name: string;
}

export interface Payout {
  id: number;
  booking_id: number;
  amount_label: string;
  status: string;
  transfer_reference: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface OwnerFinanceSummary {
  from: string;
  to: string;
  earnings_label: string;
  payouts_completed_label: string;
  pending_balance_label: string;
  deposits_held_label: string;
  paid_invoices: number;
}

export interface OwnerLedgerEntry {
  id: number;
  occurred_at: string;
  type: string;
  amount: number;
  amount_label: string;
  booking_id: number | null;
  invoice_number: string | null;
  description: string;
}

export interface OwnerStats {
  from: string;
  to: string;
  views: number;
  bookings_paid: number;
  bookings_approved: number;
  bookings_rejected: number;
  earnings_label: string;
  occupancy: { occupied: number; total: number; rate: number };
  rating: { average: number | null; count: number };
}

export interface AuditLogEntry {
  id: number;
  created_at: string;
  actor: { id: number; name: string; role: string } | null;
  action: string;
  entity_type: string;
  entity_id: number;
  changes: Record<string, [unknown, unknown]>;
}
