"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";
import { Button } from "./button";

export interface OtpInputProps {
  length?: number;
  onComplete: (code: string) => void;
  error?: string;
  disabled?: boolean;
}

/** DESIGN.md §6.4 — one hidden input catches typing/paste/SMS autofill behind 4 visual boxes. */
export function OtpInput({ length = 4, onComplete, error, disabled }: OtpInputProps) {
  const [value, setValue] = useState("");
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const firedRef = useRef(false);

  useEffect(() => {
    if (value.length === length && !firedRef.current) {
      firedRef.current = true;
      onComplete(value);
    }
    if (value.length < length) {
      firedRef.current = false;
    }
  }, [value, length, onComplete]);

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative flex gap-3">
        <input
          ref={inputRef}
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={length}
          value={value}
          disabled={disabled}
          onChange={(event) => setValue(event.target.value.replace(/\D/g, "").slice(0, length))}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          aria-label="Kode verifikasi"
          aria-invalid={error ? true : undefined}
          className="absolute inset-0 size-full cursor-text opacity-0"
        />
        {Array.from({ length }).map((_, index) => {
          const digit = value[index] ?? "";
          const isActive = focused && index === value.length;
          return (
            <span
              key={index}
              aria-hidden="true"
              className={cn(
                "tabular flex h-14 w-[52px] items-center justify-center rounded-md border text-title-lg",
                error ? "border-danger" : isActive ? "border-brand-600" : "border-neutral-400",
              )}
            >
              {digit}
            </span>
          );
        })}
      </div>
      {error && <p className="text-caption text-danger">{error}</p>}
    </div>
  );
}

/** DESIGN.md §6.4 — countdown derived from `verification_code.can_resend_at` (unix seconds). */
export function OtpResendCountdown({ canResendAt, onResend }: { canResendAt: number; onResend: () => void }) {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    const id = setInterval(() => setNow(Math.floor(Date.now() / 1000)), 1000);
    return () => clearInterval(id);
  }, []);

  const remaining = Math.max(0, canResendAt - now);
  if (remaining <= 0) {
    return (
      <Button variant="neutral" onClick={onResend}>
        Kirim ulang kode
      </Button>
    );
  }

  const minutes = Math.floor(remaining / 60);
  const seconds = remaining % 60;
  return (
    <p className="text-caption text-neutral-500">
      Kirim ulang dalam {minutes}:{String(seconds).padStart(2, "0")}
    </p>
  );
}
