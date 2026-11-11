import { Download } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { Invoice } from "@/lib/booking";
import { formatDateShort, previousDay } from "@/lib/format";
import { invoiceStatusLabel, invoiceTypeLabel } from "@/lib/labels";

export function InvoiceCard({ invoice, paying, onPay }: { invoice: Invoice; paying: boolean; onPay?: () => void }) {
  const status = invoiceStatusLabel(invoice.status);

  return (
    <article className="flex min-w-0 flex-col gap-2 rounded-lg border border-neutral-200 p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="min-w-0 break-words text-caption text-neutral-600">{invoice.number}</p>
        <div className="flex items-center gap-1">
          <Badge color={status.color}>{status.label}</Badge>
          <a
            href={`/api/invoices/${invoice.id}/pdf`}
            aria-label="Unduh PDF"
            className="relative inline-flex size-9 items-center justify-center rounded-full text-neutral-700 hover:bg-neutral-100"
          >
            <Download aria-hidden="true" size={18} strokeWidth={1.75} />
          </a>
        </div>
      </div>
      <p className="text-title-sm font-semibold text-neutral-900">{invoiceTypeLabel(invoice.type)}</p>
      <p className="text-caption text-neutral-600">
        {formatDateShort(invoice.period_start)} – {formatDateShort(previousDay(invoice.period_end))}
      </p>
      <div className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          <p className="text-price font-bold tabular-nums text-neutral-900">{invoice.total_label}</p>
          <p className="text-caption text-neutral-600">Jatuh tempo {formatDateShort(invoice.due_date)}</p>
        </div>
        {invoice.status === "unpaid" && onPay && (
          <Button loading={paying} onClick={onPay}>
            Bayar
          </Button>
        )}
      </div>
    </article>
  );
}
