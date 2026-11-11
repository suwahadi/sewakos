"use client";

import { Minus, Plus } from "lucide-react";
import { useId, useState } from "react";
import { Chip } from "@/components/ui/chip";
import { IconButton } from "@/components/ui/icon-button";
import { formatDateLong, previousDay } from "@/lib/format";
import { durationLabel, rentTypeUnit } from "@/lib/labels";
import type { RentType } from "@/lib/period";

const MAX_DURATION: Record<Exclude<RentType, "monthly">, number> = { daily: 30, weekly: 12 };

export interface DurationPickerProps {
  rentType: RentType;
  value: number;
  onChange: (value: number) => void;
  monthlyOptions: number[];
  endDate?: string;
}

export function DurationPicker({ rentType, value, onChange, monthlyOptions, endDate }: DurationPickerProps) {
  const labelId = useId();

  return (
    <div className="flex flex-col gap-1.5">
      <p id={labelId} className="text-label font-medium text-neutral-700">
        Durasi
      </p>
      <div role="group" aria-labelledby={labelId}>
        {rentType === "monthly" ? (
          <div className="flex flex-wrap gap-2">
            {monthlyOptions.map((option) => (
              <Chip key={option} active={option === value} onClick={() => onChange(option)}>
                {durationLabel("monthly", option)}
              </Chip>
            ))}
          </div>
        ) : (
          <Stepper unit={rentTypeUnit(rentType)} max={MAX_DURATION[rentType]} value={value} onChange={onChange} />
        )}
      </div>
      {endDate && <p className="text-caption text-neutral-500">Sampai {formatDateLong(previousDay(endDate))}</p>}
    </div>
  );
}

function Stepper({
  unit,
  max,
  value,
  onChange,
}: {
  unit: string;
  max: number;
  value: number;
  onChange: (value: number) => void;
}) {
  const [draft, setDraft] = useState<string | null>(null);

  function handleType(raw: string) {
    setDraft(raw);
    const typed = Number(raw);
    if (Number.isInteger(typed) && typed >= 1 && typed <= max) onChange(typed);
  }

  function step(next: number) {
    setDraft(null);
    onChange(next);
  }

  return (
    <div className="flex items-center gap-3">
      <IconButton
        aria-label="Kurangi"
        icon={<Minus aria-hidden="true" size={20} strokeWidth={1.75} />}
        className="border border-neutral-300"
        disabled={value <= 1}
        onClick={() => step(value - 1)}
      />
      <input
        type="number"
        inputMode="numeric"
        aria-label={`Jumlah ${unit}`}
        min={1}
        max={max}
        value={draft ?? String(value)}
        onChange={(event) => handleType(event.target.value)}
        onBlur={() => setDraft(null)}
        className="h-11 w-16 rounded-md border border-neutral-400 text-center text-title-md font-semibold tabular-nums text-neutral-900 focus:border-brand-600 focus:outline-none focus:ring-[3px] focus:ring-brand-100"
      />
      <IconButton
        aria-label="Tambah"
        icon={<Plus aria-hidden="true" size={20} strokeWidth={1.75} />}
        className="border border-neutral-300"
        disabled={value >= max}
        onClick={() => step(value + 1)}
      />
      <span className="text-body text-neutral-700">{unit}</span>
    </div>
  );
}
