"use client";

import { CircleAlert } from "lucide-react";
import { useId, type TextareaHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label: string;
  error?: string;
  helper?: string;
  /** Renders a "n/max" counter in the bottom-right corner, turning danger past the limit. */
  maxLength?: number;
}

export function Textarea({ label, error, helper, id, className, maxLength, value, ...props }: TextareaProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const describedBy = error ? `${inputId}-error` : helper ? `${inputId}-helper` : undefined;
  const length = typeof value === "string" ? value.length : 0;
  const overLimit = maxLength !== undefined && length > maxLength;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={inputId} className="text-label font-medium text-neutral-700">
        {label}
      </label>
      <textarea
        id={inputId}
        value={value}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={cn(
          "min-h-24 w-full resize-y rounded-md border px-3.5 py-3 text-base text-neutral-900 placeholder:text-neutral-500 md:text-body",
          "focus:border-brand-600 focus:outline-none focus:ring-[3px] focus:ring-brand-100",
          "disabled:border-neutral-300 disabled:bg-neutral-50",
          error ? "border-danger" : "border-neutral-400",
          className,
        )}
        {...props}
      />
      <div className="flex items-start justify-between gap-2">
        <div>
          {error ? (
            <p id={`${inputId}-error`} className="flex items-center gap-1 text-caption text-danger">
              <CircleAlert aria-hidden="true" size={16} strokeWidth={1.75} />
              {error}
            </p>
          ) : helper ? (
            <p id={`${inputId}-helper`} className="text-caption text-neutral-500">
              {helper}
            </p>
          ) : null}
        </div>
        {maxLength !== undefined && (
          <span className={cn("shrink-0 text-caption", overLimit ? "text-danger" : "text-neutral-500")}>
            {length}/{maxLength}
          </span>
        )}
      </div>
    </div>
  );
}
