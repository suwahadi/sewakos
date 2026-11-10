import { Check, ChevronDown } from "lucide-react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/cn";

export interface ChipProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean;
  /** Set when this chip opens a sheet (e.g. "Harga") instead of toggling by itself. */
  opensSheet?: boolean;
}

export function Chip({ active = false, opensSheet = false, className, children, ...props }: ChipProps) {
  return (
    <button
      type="button"
      aria-pressed={active}
      className={cn(
        "inline-flex h-9 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full border px-3.5 text-label font-medium",
        active ? "border-brand-600 bg-brand-50 text-brand-800" : "border-neutral-300 bg-neutral-0 text-neutral-700",
        className,
      )}
      {...props}
    >
      {active && <Check aria-hidden="true" size={16} strokeWidth={1.75} />}
      {children}
      {opensSheet && <ChevronDown aria-hidden="true" size={16} strokeWidth={1.75} />}
    </button>
  );
}

/** DESIGN.md §0.1-E — a horizontally scrollable row of chips, snapped and scrollbar-free. */
export function ChipRow({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "flex gap-2 overflow-x-auto px-4 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden",
        className,
      )}
      style={{ scrollSnapType: "x proximity" }}
    >
      {children}
    </div>
  );
}
