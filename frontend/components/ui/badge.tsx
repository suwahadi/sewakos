import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import type { BadgeColor } from "@/lib/labels";

const COLOR_CLASSES: Record<BadgeColor | "brand", string> = {
  success: "bg-success-bg text-success",
  warning: "bg-warning-bg text-warning",
  danger: "bg-danger-bg text-danger",
  info: "bg-info-bg text-info",
  neutral: "bg-neutral-100 text-neutral-600",
  brand: "bg-brand-600 text-neutral-0",
};

export interface BadgeProps {
  color: BadgeColor | "brand";
  icon?: ReactNode;
  className?: string;
  children: ReactNode;
}

/** DESIGN.md §6.12 — a badge always carries text; color is only a secondary cue. */
export function Badge({ color, icon, className, children }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex h-[22px] items-center gap-1 rounded-sm px-2 text-caption-strong",
        COLOR_CLASSES[color],
        className,
      )}
    >
      {icon}
      {children}
    </span>
  );
}

/** DESIGN.md §6.12 "Hitungan" — e.g. the unread-notifications count on a bell icon. */
export function CountBadge({ count, className }: { count: number; className?: string }) {
  if (count <= 0) return null;
  return (
    <span
      className={cn(
        "inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-brand-600 px-1 text-caption-strong text-neutral-0",
        className,
      )}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
}
