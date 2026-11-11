import type { OwnerBooking, OwnerProperty } from "./owner";

export const ADMIN_PAGE_SIZE = 20;

export interface AdminArea {
  id: number;
  name: string;
  slug: string;
  type: "city" | "campus" | "mrt";
  is_top: boolean;
  latitude: number;
  longitude: number;
  place_id: string | null;
}

export interface AdminUser {
  id: number;
  name: string;
  role: string;
  phone_number: string;
  email: string | null;
  is_verified: boolean;
  suspended_at: string | null;
  created_at: string;
}

export interface AdminProperty extends OwnerProperty {
  slug: string;
  created_at: string;
  owner: { id: number; name: string; phone_number: string };
}

export interface AdminBooking extends OwnerBooking {
  admin_fee_label: string;
  created_at: string;
}

export interface AdminBankAccount {
  bank_name: string;
  account_number: string;
  account_name: string;
}

export interface AdminRefund {
  id: number;
  booking_id: number;
  tenant: { id: number; name: string };
  amount_label: string;
  reason: string;
  status: string;
  bank_account: AdminBankAccount | null;
  transfer_reference: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface AdminPayout {
  id: number;
  booking_id: number;
  owner: { id: number; name: string };
  amount_label: string;
  status: string;
  bank_account: AdminBankAccount | null;
  transfer_reference: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface AdminInvoice {
  id: number;
  number: string;
  booking_id: number;
  type: string;
  status: string;
  total_label: string;
  due_date: string;
}

const AREA_TYPE_LABEL: Record<string, string> = { city: "Kota", campus: "Kampus", mrt: "Stasiun MRT" };
const REFUND_REASON_LABEL: Record<string, string> = {
  owner_rejected: "Ditolak pemilik",
  owner_timeout: "Pemilik tidak merespons",
  tenant_cancelled: "Dibatalkan penyewa",
  stock_unavailable: "Kamar habis",
  invoice_void: "Tagihan dibatalkan",
};
const ROLE_LABEL: Record<string, string> = { tenant: "Penyewa", owner: "Pemilik", admin: "Admin" };

export const areaTypeLabel = (type: string) => AREA_TYPE_LABEL[type] ?? type;
export const refundReasonLabel = (reason: string) => REFUND_REASON_LABEL[reason] ?? reason;
export const roleLabel = (role: string) => ROLE_LABEL[role] ?? role;
