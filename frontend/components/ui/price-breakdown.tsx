export interface PriceBreakdownProps {
  priceLabel: string;
  depositLabel: string;
  adminFeeLabel?: string;
  flashSaleDiscountLabel?: string;
  remainingPaymentLabel?: string;
  totalLabel: string;
  nextMonthLabel?: string;
}

export function PriceBreakdown({
  priceLabel,
  depositLabel,
  adminFeeLabel,
  flashSaleDiscountLabel,
  remainingPaymentLabel,
  totalLabel,
  nextMonthLabel,
}: PriceBreakdownProps) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-neutral-200 p-4">
      <Row label="Sewa" value={priceLabel} />
      <div>
        <Row label="Deposit" value={depositLabel} />
        <p className="text-caption text-neutral-600">Dikembalikan pemilik di akhir sewa</p>
      </div>
      {adminFeeLabel && <Row label="Biaya admin" value={adminFeeLabel} />}
      {flashSaleDiscountLabel && (
        <Row label="Diskon promo" value={`− ${flashSaleDiscountLabel}`} valueClassName="text-brand-700" />
      )}
      <div className="border-t border-neutral-200 pt-3">
        {remainingPaymentLabel && <p className="text-body text-neutral-500 line-through">{remainingPaymentLabel}</p>}
        <Row label="Total bayar sekarang" value={totalLabel} strong />
        {nextMonthLabel && <p className="text-caption text-neutral-600">{nextMonthLabel}</p>}
      </div>
    </div>
  );
}

function Row({
  label,
  value,
  strong,
  valueClassName,
}: {
  label: string;
  value: string;
  strong?: boolean;
  valueClassName?: string;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <p className="text-body text-neutral-700">{label}</p>
      <p className={strong ? "text-price-lg font-bold tabular-nums text-neutral-900" : `text-body tabular-nums text-neutral-900 ${valueClassName ?? ""}`}>
        {value}
      </p>
    </div>
  );
}
