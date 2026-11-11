const WIB_TIME_ZONE = "Asia/Jakarta";

export function formatRupiah(amount: number): string {
  return `Rp${amount.toLocaleString("id-ID")}`;
}

export function formatDateShort(dateInput: string | Date): string {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    timeZone: WIB_TIME_ZONE,
  }).format(new Date(dateInput));
}

export function formatDateLong(dateInput: string | Date): string {
  return new Intl.DateTimeFormat("id-ID", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone: WIB_TIME_ZONE,
  }).format(new Date(dateInput));
}

export function formatTimeWIB(dateInput: string | Date): string {
  const time = new Intl.DateTimeFormat("id-ID", {
    hour: "2-digit",
    minute: "2-digit",
    timeZone: WIB_TIME_ZONE,
  }).format(new Date(dateInput));
  return `${time} WIB`;
}

const RELATIVE_UNITS: { unit: Intl.RelativeTimeFormatUnit; seconds: number }[] = [
  { unit: "year", seconds: 31536000 },
  { unit: "month", seconds: 2592000 },
  { unit: "day", seconds: 86400 },
  { unit: "hour", seconds: 3600 },
  { unit: "minute", seconds: 60 },
  { unit: "second", seconds: 1 },
];

/** > 7 hari ditampilkan sebagai tanggal (DESIGN.md §8.3), bukan waktu relatif. */
export function formatRelativeTime(dateInput: string | Date, now: Date = new Date()): string {
  const target = new Date(dateInput);
  const diffSeconds = (target.getTime() - now.getTime()) / 1000;
  if (Math.abs(diffSeconds) > 7 * 86400) {
    return formatDateShort(target);
  }

  const rtf = new Intl.RelativeTimeFormat("id", { numeric: "auto" });
  for (const { unit, seconds } of RELATIVE_UNITS) {
    if (Math.abs(diffSeconds) >= seconds || unit === "second") {
      return rtf.format(Math.round(diffSeconds / seconds), unit);
    }
  }
  return rtf.format(0, "second");
}

export function formatRating(average: number): string {
  return average.toLocaleString("id-ID", { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

export function formatPercent(rate: number): string {
  return `${Math.round(rate * 100)}%`;
}

export function formatPhoneDisplay(phone: string): string {
  const groups: string[] = [];
  for (let i = 0; i < phone.length; i += 4) {
    groups.push(phone.slice(i, i + 4));
  }
  return groups.join(" ");
}

/** Mengurangi 1 hari kalender dari tanggal `YYYY-MM-DD` (untuk `end_date`/`period_end` yang eksklusif). */
export function previousDay(dateOnly: string): string {
  const [year, month, day] = dateOnly.split("-").map(Number);
  const date = new Date(Date.UTC(year!, month! - 1, day!));
  date.setUTCDate(date.getUTCDate() - 1);
  const y = date.getUTCFullYear();
  const m = String(date.getUTCMonth() + 1).padStart(2, "0");
  const d = String(date.getUTCDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export function digitsOnly(value: string): string {
  return value.replace(/\D/g, "");
}

export function formatThousands(digits: string): string {
  return digits === "" ? "" : Number(digits).toLocaleString("id-ID");
}
