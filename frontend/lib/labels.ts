export type BadgeColor = "success" | "warning" | "danger" | "info" | "neutral";

export interface StatusLabel {
  label: string;
  color: BadgeColor;
}

function lookup(table: Record<string, StatusLabel>, value: string): StatusLabel {
  return table[value] ?? { label: value, color: "neutral" };
}

const BOOKING_STATUS: Record<string, StatusLabel> = {
  pending: { label: "Menunggu pembayaran", color: "warning" },
  paid: { label: "Menunggu konfirmasi pemilik", color: "info" },
  approved: { label: "Aktif", color: "success" },
  completed: { label: "Selesai", color: "neutral" },
  terminated: { label: "Dihentikan", color: "danger" },
  rejected: { label: "Ditolak", color: "danger" },
  cancelled: { label: "Dibatalkan", color: "neutral" },
};

const INVOICE_STATUS: Record<string, StatusLabel> = {
  unpaid: { label: "Belum dibayar", color: "warning" },
  paid: { label: "Lunas", color: "success" },
  void: { label: "Dibatalkan", color: "neutral" },
};

const PAYMENT_STATUS: Record<string, StatusLabel> = {
  pending: { label: "Menunggu pembayaran", color: "warning" },
  settlement: { label: "Berhasil", color: "success" },
  expire: { label: "Kedaluwarsa", color: "neutral" },
  cancel: { label: "Dibatalkan", color: "danger" },
  deny: { label: "Ditolak", color: "danger" },
  failure: { label: "Gagal", color: "danger" },
};

const REFUND_STATUS: Record<string, StatusLabel> = {
  awaiting_account: { label: "Perlu rekening", color: "warning" },
  ready: { label: "Diproses", color: "info" },
  completed: { label: "Selesai", color: "success" },
};

const PAYOUT_STATUS: Record<string, StatusLabel> = {
  pending: { label: "Menunggu transfer", color: "warning" },
  completed: { label: "Sudah ditransfer", color: "success" },
};

const MODERATION_STATUS: Record<string, StatusLabel> = {
  pending: { label: "Sedang ditinjau", color: "info" },
  approved: { label: "Tayang", color: "success" },
  rejected: { label: "Ditolak", color: "danger" },
  suspended: { label: "Dinonaktifkan", color: "danger" },
};

export const bookingStatusLabel = (status: string) => lookup(BOOKING_STATUS, status);
export const invoiceStatusLabel = (status: string) => lookup(INVOICE_STATUS, status);
export const paymentStatusLabel = (status: string) => lookup(PAYMENT_STATUS, status);
export const refundStatusLabel = (status: string) => lookup(REFUND_STATUS, status);
export const payoutStatusLabel = (status: string) => lookup(PAYOUT_STATUS, status);
export const moderationStatusLabel = (status: string) => lookup(MODERATION_STATUS, status);

const GENDER_LABEL: Record<string, string> = {
  putra: "Putra",
  putri: "Putri",
  campur: "Campur",
};

export function genderLabel(gender: string): string {
  return GENDER_LABEL[gender] ?? gender;
}

const NOTIFICATION_CATEGORY_LABEL: Record<string, string> = {
  booking: "Booking",
  payment: "Pembayaran",
  promo: "Promo",
  chat: "Chat",
};

export function notificationCategoryLabel(category: string): string {
  return NOTIFICATION_CATEGORY_LABEL[category] ?? category;
}
