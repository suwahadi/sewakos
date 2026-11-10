import { cn } from "@/lib/cn";
import { genderLabel } from "@/lib/labels";

const GENDER_CLASSES: Record<string, string> = {
  putra: "bg-putra-bg text-putra",
  putri: "bg-putri-bg text-putri",
  campur: "bg-campur-bg text-campur",
};

/** DESIGN.md §3.4 — gender badge always shows the label as text; color is a secondary cue. */
export function GenderBadge({ gender, className }: { gender: string; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex h-[22px] items-center rounded-sm px-2 text-caption-strong",
        GENDER_CLASSES[gender] ?? "bg-neutral-100 text-neutral-600",
        className,
      )}
    >
      {genderLabel(gender)}
    </span>
  );
}
