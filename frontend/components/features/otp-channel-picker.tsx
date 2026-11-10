"use client";

import { cn } from "@/lib/cn";

export type OtpChannel = "0" | "1";

// The official WhatsApp brand SVG isn't available yet (PRD.md §8 #7), so DESIGN.md §0.1-C's own
// fallback applies here: show the brand name as plain text, no substitute icon. Both options drop
// their icon together to keep the 50:50 pair consistent (§0.1-B) — once the asset is provided, add
// the lucide-react SMS icon back alongside the real WhatsApp logo for both options.

export interface OtpChannelPickerProps {
  value: OtpChannel;
  onChange: (value: OtpChannel) => void;
}

export function OtpChannelPicker({ value, onChange }: OtpChannelPickerProps) {
  return (
    <div role="radiogroup" aria-label="Kirim kode lewat" className="grid grid-cols-2 gap-3">
      <OtpChannelOption selected={value === "0"} onClick={() => onChange("0")} label="SMS" />
      <OtpChannelOption selected={value === "1"} onClick={() => onChange("1")} label="WhatsApp" />
    </div>
  );
}

function OtpChannelOption({
  selected,
  onClick,
  label,
}: {
  selected: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onClick}
      className={cn(
        "flex h-[52px] items-center justify-center gap-2 rounded-md border text-label font-medium transition-colors duration-[var(--duration-fast)] ease-standard",
        selected ? "border-2 border-brand-600 bg-brand-50 text-brand-800" : "border-neutral-300 text-neutral-700",
      )}
    >
      {label}
    </button>
  );
}
