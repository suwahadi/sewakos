export interface StatCardProps {
  label: string;
  value: string;
  caption?: string;
}

export function StatCard({ label, value, caption }: StatCardProps) {
  return (
    <div className="flex h-full flex-col gap-1 rounded-lg bg-neutral-50 p-4">
      <p className="text-caption text-neutral-600">{label}</p>
      <p className="text-title-lg font-bold tabular-nums text-neutral-900">{value}</p>
      <p className="text-caption text-neutral-600">{caption ?? " "}</p>
    </div>
  );
}
