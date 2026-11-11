export interface StatCardProps {
  label: string;
  value: string;
  caption?: string;
}

/** DESIGN.md §6.44 — caption is always rendered (as a non-breaking space when absent) so cards in the same row stay equal height. */
export function StatCard({ label, value, caption }: StatCardProps) {
  return (
    <div className="flex h-full flex-col gap-1 rounded-lg bg-neutral-50 p-4">
      <p className="text-caption text-neutral-600">{label}</p>
      <p className="text-title-lg font-bold tabular-nums text-neutral-900">{value}</p>
      <p className="text-caption text-neutral-600">{caption ?? " "}</p>
    </div>
  );
}
