export type RentType = "monthly" | "weekly" | "daily";

export function todayWIB(): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(new Date());
}

function toDateOnly(date: Date): string {
  return date.toISOString().slice(0, 10);
}

function parseDateOnly(dateOnly: string): Date {
  const [year, month, day] = dateOnly.split("-").map(Number);
  return new Date(Date.UTC(year!, month! - 1, day!));
}

function addMonthsClamped(start: Date, months: number): Date {
  const target = new Date(Date.UTC(start.getUTCFullYear(), start.getUTCMonth() + months, 1));
  const lastDay = new Date(Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0)).getUTCDate();
  target.setUTCDate(Math.min(start.getUTCDate(), lastDay));
  return target;
}

export function rentalEndDate(startDate: string, rentType: RentType, duration: number): string {
  const start = parseDateOnly(startDate);
  if (rentType === "monthly") return toDateOnly(addMonthsClamped(start, duration));
  start.setUTCDate(start.getUTCDate() + duration * (rentType === "weekly" ? 7 : 1));
  return toDateOnly(start);
}
