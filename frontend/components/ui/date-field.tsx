"use client";

import { useId } from "react";
import { cn } from "@/lib/cn";
import { formatDateLong } from "@/lib/format";

function todayWIB(): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(new Date());
}

export interface DateFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  min?: string;
  error?: string;
  id?: string;
  name?: string;
  required?: boolean;
}

/** DESIGN.md §6.10 — `min` defaults to today in WIB; shows the selected date spelled out below. */
export function DateField({ label, value, onChange, min, error, id, name, required }: DateFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const minDate = min ?? todayWIB();
  const describedBy = error ? `${inputId}-error` : undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={inputId} className="text-label font-medium text-neutral-700">
        {label}
      </label>
      <input
        id={inputId}
        name={name}
        type="date"
        required={required}
        min={minDate}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={cn(
          "h-12 w-full rounded-md border px-3.5 text-base text-neutral-900 md:text-body",
          "focus:border-brand-600 focus:outline-none focus:ring-[3px] focus:ring-brand-100",
          error ? "border-danger" : "border-neutral-400",
        )}
      />
      {value && <p className="text-caption text-neutral-500">{formatDateLong(value)}</p>}
      {error && (
        <p id={`${inputId}-error`} className="text-caption text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
