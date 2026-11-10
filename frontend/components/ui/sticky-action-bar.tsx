import { Button } from "./button";

export interface StickyActionBarProps {
  leadingLabel: string;
  price: string;
  priceUnit?: string;
  actionLabel: string;
  onAction: () => void;
  disabled?: boolean;
}

export function StickyActionBar({ leadingLabel, price, priceUnit, actionLabel, onAction, disabled }: StickyActionBarProps) {
  return (
    <div
      className="fixed inset-x-0 bottom-0 z-[var(--z-sticky)] flex items-center justify-between gap-3 border-t border-neutral-200 bg-neutral-0 px-4 py-3 shadow-raised lg:sticky lg:top-20 lg:bottom-auto lg:rounded-lg lg:border"
      style={{ paddingBottom: "calc(env(safe-area-inset-bottom) + 12px)" }}
    >
      <div className="min-w-0">
        <p className="text-caption text-neutral-600">{leadingLabel}</p>
        <p className="truncate text-price font-bold text-neutral-900">
          {price}
          {priceUnit && <span className="text-body font-normal text-neutral-600"> {priceUnit}</span>}
        </p>
      </div>
      <Button size="lg" onClick={onAction} disabled={disabled}>
        {actionLabel}
      </Button>
    </div>
  );
}
