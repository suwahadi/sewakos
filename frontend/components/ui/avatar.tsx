import { cn } from "@/lib/cn";

function initials(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((word) => word[0] ?? "")
    .join("")
    .toUpperCase();
}

/** API provides no profile photo, so DESIGN.md §6.13 always renders initials. */
export function Avatar({ name, size = "list" }: { name: string; size?: "list" | "profile" }) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full bg-brand-100 font-semibold text-brand-800",
        size === "profile" ? "size-16 text-title-lg" : "size-10 text-label",
      )}
    >
      {initials(name)}
    </span>
  );
}
