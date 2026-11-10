import { cn } from "@/lib/cn";

/** DESIGN.md §6.14 — size it to match the real content exactly via className. */
export function Skeleton({ className }: { className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "block animate-shimmer bg-neutral-100 bg-[length:200%_100%] bg-gradient-to-r from-neutral-100 via-neutral-50 to-neutral-100",
        className,
      )}
    />
  );
}
