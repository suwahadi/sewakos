import { formatMonthYear } from "@/lib/format";

export function ownerInitials(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");
}

export function OwnerSummary({ name, activeSince }: { name: string; activeSince: string }) {
  return (
    <section className="flex items-center gap-3 border-t border-neutral-200 pt-6">
      <span
        aria-hidden="true"
        className="flex aspect-square shrink-0 items-center justify-center self-stretch rounded-full bg-brand-100 text-label font-semibold text-brand-800"
      >
        {ownerInitials(name)}
      </span>
      <div className="flex min-w-0 flex-col">
        <p className="truncate text-body-strong text-neutral-900">{name}</p>
        <p className="truncate text-caption text-neutral-600">Pemilik Kos · Aktif sejak {formatMonthYear(activeSince)}</p>
      </div>
    </section>
  );
}
