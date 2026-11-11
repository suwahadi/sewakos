"use client";

import { CircleAlert, Eye, EyeOff } from "lucide-react";
import { useId, useState, type InputHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  helper?: string;
  prefix?: string;
}

export function TextField({ label, error, helper, prefix, id, className, type = "text", ...props }: TextFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const [showPassword, setShowPassword] = useState(false);
  const isPassword = type === "password";
  const resolvedType = isPassword ? (showPassword ? "text" : "password") : type;
  const describedBy = error ? `${inputId}-error` : helper ? `${inputId}-helper` : undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={inputId} className="text-label font-medium text-neutral-700">
        {label}
      </label>
      <div className="relative">
        {prefix && (
          <span aria-hidden="true" className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-body text-neutral-600">
            {prefix}
          </span>
        )}
        <input
          id={inputId}
          type={resolvedType}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cn(
            "h-12 w-full rounded-md border px-3.5 text-base text-neutral-900 placeholder:text-neutral-500 md:text-body",
            "focus:border-brand-600 focus:outline-none focus:ring-[3px] focus:ring-brand-100",
            "disabled:border-neutral-300 disabled:bg-neutral-50",
            error ? "border-danger" : "border-neutral-400",
            isPassword && "pr-11",
            prefix && "pl-10",
            className,
          )}
          {...props}
        />
        {isPassword && (
          <button
            type="button"
            onClick={() => setShowPassword((value) => !value)}
            aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
            className="absolute right-1 top-1/2 inline-flex size-9 -translate-y-1/2 items-center justify-center rounded-md text-neutral-600 hover:bg-neutral-100"
          >
            {showPassword ? (
              <EyeOff aria-hidden="true" size={20} strokeWidth={1.75} />
            ) : (
              <Eye aria-hidden="true" size={20} strokeWidth={1.75} />
            )}
          </button>
        )}
      </div>
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
  );
}
