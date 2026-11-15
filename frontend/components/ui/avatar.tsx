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
export function Avatar({
  name,
  size = "list",
  tone = "soft",
}: {
  name: string;
  size?: "list" | "profile";
  tone?: "soft" | "solid";
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full font-semibold",
        tone === "solid" ? "bg-brand-600 text-neutral-0" : "bg-brand-100 text-brand-800",
        size === "profile" ? "size-16 text-title-lg" : "size-10 text-label",
      )}
    >
      {initials(name)}
    </span>
  );
}
