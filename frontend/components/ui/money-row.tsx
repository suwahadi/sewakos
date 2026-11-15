import type { ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import { formatDateShort, formatRupiah } from "@/lib/format";

export interface MoneyRowProps {
  description: string;
  detail?: ReactNode;
  date: string;
  amount: number;
  isNote?: boolean;
}

export function MoneyRow({ description, detail, date, amount, isNote }: MoneyRowProps) {
  const negative = amount < 0;
  return (
    <div className="flex items-center justify-between gap-3 py-3">
      <div className="min-w-0">
        <p className="break-words text-body text-neutral-900">{description}</p>
        {detail && <div className="text-caption text-neutral-600">{detail}</div>}
        <p className="text-caption text-neutral-600">{formatDateShort(date)}</p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {isNote && <Badge color="neutral">Catatan</Badge>}
        <p className={`text-body-strong tabular-nums ${negative ? "text-neutral-700" : "text-neutral-900"}`}>
          {negative ? "− " : ""}
          {formatRupiah(Math.abs(amount))}
        </p>
      </div>
    </div>
  );
}
