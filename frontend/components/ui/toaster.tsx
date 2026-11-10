"use client";

import { CircleAlert, CircleCheck, Info, TriangleAlert, X } from "lucide-react";
import { useEffect, useRef, useSyncExternalStore } from "react";
import { cn } from "@/lib/cn";
import { getToastsSnapshot, subscribeToasts, toast, type ToastItem, type ToastVariant } from "@/lib/toast";

const VARIANT_ICON: Record<ToastVariant, typeof Info> = {
  success: CircleCheck,
  error: CircleAlert,
  warning: TriangleAlert,
  info: Info,
};

// *-on-dark tokens (DESIGN.md §3.3), for icons on the toast's dark neutral-800 background.
const VARIANT_ICON_CLASS: Record<ToastVariant, string> = {
  success: "text-success-on-dark",
  error: "text-danger-on-dark",
  warning: "text-warning-on-dark",
  info: "text-info-on-dark",
};

const MAX_VISIBLE = 3;

export function Toaster() {
  const items = useSyncExternalStore(subscribeToasts, getToastsSnapshot, getToastsSnapshot);
  const visible = items.slice(-MAX_VISIBLE);

  return (
    <section
      aria-label="Notifikasi"
      className="pointer-events-none fixed inset-x-0 bottom-[calc(64px+env(safe-area-inset-bottom)+12px)] z-[var(--z-toast)] flex flex-col items-center gap-2 lg:inset-x-auto lg:right-6 lg:top-6 lg:bottom-auto lg:items-end"
    >
      {visible.map((item) => (
        <ToastCard key={item.id} toast={item} />
      ))}
    </section>
  );
}

function ToastCard({ toast: item }: { toast: ToastItem }) {
  const remainingRef = useRef(item.duration);
  const startedAtRef = useRef<number | null>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    function clear() {
      if (timeoutRef.current !== null) clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    function start() {
      if (!Number.isFinite(remainingRef.current)) return;
      startedAtRef.current = Date.now();
      timeoutRef.current = setTimeout(() => toast.dismiss(item.id), remainingRef.current);
    }
    start();
    return clear;
  }, [item.id]);

  function pause() {
    if (timeoutRef.current === null || startedAtRef.current === null) return;
    const elapsed = Date.now() - startedAtRef.current;
    remainingRef.current = Math.max(0, remainingRef.current - elapsed);
    clearTimeout(timeoutRef.current);
    timeoutRef.current = null;
  }

  function resume() {
    if (timeoutRef.current !== null || !Number.isFinite(remainingRef.current)) return;
    startedAtRef.current = Date.now();
    timeoutRef.current = setTimeout(() => toast.dismiss(item.id), remainingRef.current);
  }

  const Icon = VARIANT_ICON[item.variant];

  return (
    <div
      role={item.variant === "error" ? "alert" : "status"}
      aria-live={item.variant === "error" ? undefined : "polite"}
      onMouseEnter={pause}
      onMouseLeave={resume}
      onFocus={pause}
      onBlur={resume}
      className="animate-toast-in pointer-events-auto flex w-[min(100%-32px,400px)] items-start gap-3 rounded-lg bg-neutral-800 p-3 shadow-overlay"
    >
      <Icon aria-hidden="true" size={20} strokeWidth={1.75} className={cn("mt-0.5 shrink-0", VARIANT_ICON_CLASS[item.variant])} />
      <div className="min-w-0 flex-1">
        <p className="text-body-strong text-neutral-0">{item.title}</p>
        {item.description && <p className="text-caption text-neutral-300">{item.description}</p>}
        {item.action && (
          <button
            type="button"
            onClick={item.action.onClick}
            className="mt-1 text-label font-semibold text-brand-300"
          >
            {item.action.label}
          </button>
        )}
      </div>
      <button type="button" aria-label="Tutup notifikasi" onClick={() => toast.dismiss(item.id)} className="shrink-0 text-neutral-300">
        <X aria-hidden="true" size={20} strokeWidth={1.75} />
      </button>
    </div>
  );
}
