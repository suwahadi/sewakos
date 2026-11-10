import { ChevronDown } from "lucide-react";
import { useId, type SelectHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label: string;
  error?: string;
  helper?: string;
}

export function Select({ label, error, helper, id, className, children, ...props }: SelectProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const describedBy = error ? `${inputId}-error` : helper ? `${inputId}-helper` : undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={inputId} className="text-label font-medium text-neutral-700">
        {label}
      </label>
      <div className="relative">
        <select
          id={inputId}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cn(
            "h-12 w-full appearance-none rounded-md border bg-neutral-0 px-3.5 pr-10 text-base text-neutral-900 md:text-body",
            "focus:border-brand-600 focus:outline-none focus:ring-[3px] focus:ring-brand-100",
            "disabled:border-neutral-300 disabled:bg-neutral-50",
            error ? "border-danger" : "border-neutral-400",
            className,
          )}
          {...props}
        >
          {children}
        </select>
        <ChevronDown
          aria-hidden="true"
          size={20}
          strokeWidth={1.75}
          className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-neutral-600"
        />
      </div>
      {error ? (
        <p id={`${inputId}-error`} className="text-caption text-danger">
          {error}
        </p>
      ) : helper ? (
        <p id={`${inputId}-helper`} className="text-caption text-neutral-500">
          {helper}
        </p>
      ) : null}
    </div>
  );
}
