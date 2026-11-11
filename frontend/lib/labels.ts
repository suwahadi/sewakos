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

const RENT_TYPE_LABEL: Record<string, string> = {
  monthly: "Bulanan",
  weekly: "Mingguan",
  daily: "Harian",
};

const RENT_TYPE_UNIT: Record<string, string> = {
  monthly: "bulan",
  weekly: "minggu",
  daily: "hari",
};

export function rentTypeLabel(rentType: string): string {
  return RENT_TYPE_LABEL[rentType] ?? rentType;
}

export function rentTypeUnit(rentType: string): string {
  return RENT_TYPE_UNIT[rentType] ?? rentType;
}

export function durationLabel(rentType: string, duration: number): string {
  return `${duration} ${rentTypeUnit(rentType)}`;
}

const INVOICE_TYPE_LABEL: Record<string, string> = {
  initial: "Pembayaran pertama",
  renewal: "Tagihan bulanan",
  extension: "Perpanjangan",
};

export function invoiceTypeLabel(type: string): string {
  return INVOICE_TYPE_LABEL[type] ?? type;
}

const LEDGER_TYPE_LABEL: Record<string, string> = {
  payment_received: "Tagihan dibayar",
  admin_fee_earned: "Biaya admin",
  owner_earning: "Pendapatan pemilik",
  payout_completed: "Payout ditransfer",
  refund_due: "Refund dibuat",
  refund_completed: "Refund ditransfer",
  admin_fee_reversed: "Biaya admin dikembalikan",
  deposit_returned: "Deposit dikembalikan",
};

export function ledgerTypeLabel(type: string): string {
  return LEDGER_TYPE_LABEL[type] ?? type;
}

const RECONCILIATION_KIND_LABEL: Record<string, string> = {
  status_mismatch: "Status tidak cocok",
  amount_mismatch: "Nominal tidak cocok",
  not_found_at_midtrans: "Tidak ditemukan di Midtrans",
};

export function reconciliationKindLabel(kind: string): string {
  return RECONCILIATION_KIND_LABEL[kind] ?? kind;
}

const RECONCILIATION_STATUS: Record<string, StatusLabel> = {
  open: { label: "Terbuka", color: "warning" },
  resolved: { label: "Selesai", color: "success" },
};

export const reconciliationStatusLabel = (status: string) => lookup(RECONCILIATION_STATUS, status);
