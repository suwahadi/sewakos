export type ToastVariant = "success" | "error" | "warning" | "info";

export interface ToastAction {
  label: string;
  onClick: () => void;
}

export interface ToastOptions {
  /** Same id replaces the previous toast instead of stacking a duplicate. */
  id?: string;
  description?: string;
  /** ms; Infinity = stays until dismissed. Defaults per variant (DESIGN.md §6.16). */
  duration?: number;
  action?: ToastAction;
}

export interface ToastItem {
  id: string;
  variant: ToastVariant;
  title: string;
  description?: string;
  duration: number;
  action?: ToastAction;
}

const DEFAULT_DURATION: Record<ToastVariant, number> = {
  success: 4000,
  info: 4000,
  warning: 5000,
  error: 6000,
};

let toasts: ToastItem[] = [];
const listeners = new Set<() => void>();
let seq = 0;

function emit() {
  for (const listener of listeners) listener();
}

function push(variant: ToastVariant, title: string, options: ToastOptions = {}): string {
  seq += 1;
  const id = options.id ?? `toast-${seq}`;
  const duration = options.action ? Infinity : (options.duration ?? DEFAULT_DURATION[variant]);
  const item: ToastItem = { id, variant, title, description: options.description, duration, action: options.action };
  toasts = [...toasts.filter((existing) => existing.id !== id), item];
  emit();
  return id;
}

function dismiss(id: string) {
  if (!toasts.some((item) => item.id === id)) return;
  toasts = toasts.filter((item) => item.id !== id);
  emit();
}

export const toast = {
  success: (title: string, options?: ToastOptions) => push("success", title, options),
  error: (title: string, options?: ToastOptions) => push("error", title, options),
  warning: (title: string, options?: ToastOptions) => push("warning", title, options),
  info: (title: string, options?: ToastOptions) => push("info", title, options),
  dismiss,
};

export function subscribeToasts(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getToastsSnapshot(): ToastItem[] {
  return toasts;
}

/** Test-only: clears all toasts since the store is module-level singleton state. */
export function resetToastsForTests() {
  toasts = [];
  emit();
}
