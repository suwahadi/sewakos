import { Avatar } from "@/components/ui/avatar";
import { formatMonthYear } from "@/lib/format";

export function OwnerSummary({ name, activeSince }: { name: string; activeSince: string }) {
  return (
    <section className="-mx-4 mb-2.5 flex items-center gap-3 border-y border-neutral-200 bg-neutral-50 px-4 py-5">
      <Avatar name={name} tone="solid" />
      <div className="flex min-w-0 flex-col">
        <p className="truncate text-body-strong text-neutral-900">{name}</p>
        <p className="truncate text-caption text-neutral-600">Pemilik Kos · Aktif sejak {formatMonthYear(activeSince)}</p>
      </div>
    </section>
  );
}
