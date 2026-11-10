import { Check } from "lucide-react";
import { useId, type InputHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/cn";

export interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
  label: ReactNode;
}

export function Checkbox({ label, id, className, ...props }: CheckboxProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;

  return (
    <label htmlFor={inputId} className={cn("flex min-h-11 cursor-pointer items-center gap-3", className)}>
      <span className="relative inline-flex size-5 shrink-0 items-center justify-center">
        <input id={inputId} type="checkbox" className="peer absolute inset-0 size-5 cursor-pointer appearance-none" {...props} />
        <span
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 rounded-sm border border-neutral-400 peer-checked:border-brand-600 peer-checked:bg-brand-600 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-brand-600"
        />
        <Check
          aria-hidden="true"
          size={16}
          strokeWidth={2.5}
          className="pointer-events-none absolute text-neutral-0 opacity-0 peer-checked:opacity-100"
        />
      </span>
      <span className="text-body text-neutral-900">{label}</span>
    </label>
  );
}
