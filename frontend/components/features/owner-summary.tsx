import { Avatar } from "@/components/ui/avatar";
import { formatMonthYear } from "@/lib/format";

export function OwnerSummary({ name, activeSince }: { name: string; activeSince: string }) {
  return (
    <section className="flex items-center gap-3 border-t border-neutral-200 pt-6">
      <Avatar name={name} />
      <div className="flex min-w-0 flex-col">
        <p className="truncate text-body-strong text-neutral-900">{name}</p>
        <p className="truncate text-caption text-neutral-600">Pemilik Kos · Aktif sejak {formatMonthYear(activeSince)}</p>
      </div>
    </section>
  );
}
