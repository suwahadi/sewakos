"use client";

import { useEffect, useState } from "react";

const SECOND = 1000;
const MINUTE = 60 * SECOND;
export const HOUR = 60 * MINUTE;

export function formatRemaining(ms: number): string {
  if (ms >= HOUR) return `${Math.floor(ms / HOUR)} jam ${Math.floor((ms % HOUR) / MINUTE)} menit`;
  const seconds = Math.max(0, Math.floor(ms / SECOND));
  return `${String(Math.floor(seconds / 60)).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`;
}

function nextTickDelay(remaining: number): number {
  const unit = remaining >= HOUR ? MINUTE : SECOND;
  return remaining % unit || unit;
}

export function useRemainingMs(deadlineMs: number): number {
  const [now, setNow] = useState(() => Date.now());
  const remaining = deadlineMs - now;

  useEffect(() => {
    if (remaining <= 0) return;
    const timer = setTimeout(() => setNow(Date.now()), nextTickDelay(remaining));
    return () => clearTimeout(timer);
  }, [remaining]);

  return remaining;
}
