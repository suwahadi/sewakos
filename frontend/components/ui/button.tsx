import { cloneElement, forwardRef, isValidElement, type ButtonHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/cn";
import { Spinner } from "./spinner";

export type ButtonVariant = "success" | "danger" | "neutral";
export type ButtonSize = "sm" | "md" | "lg";

const VARIANT_CLASSES: Record<ButtonVariant, string> = {
  success: "border-success bg-success text-neutral-0 hover:brightness-92 active:brightness-85",
  danger: "border-danger bg-danger text-neutral-0 hover:brightness-92 active:brightness-85",
  neutral: "border-neutral-400 bg-neutral-0 text-neutral-900 hover:bg-neutral-50 active:bg-neutral-100",
};

const SIZE_CLASSES: Record<ButtonSize, string> = {
  sm: "h-9 px-3 text-label font-semibold",
  md: "h-11 px-4 text-body-strong",
  lg: "h-[52px] px-5 text-body-strong",
};

const ICON_SIZE: Record<ButtonSize, number> = { sm: 16, md: 20, lg: 20 };

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  icon?: ReactNode;
  fullWidth?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "success", size = "md", loading = false, icon, fullWidth, disabled, className, children, ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      type={props.type ?? "button"}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-md border transition-[background-color,filter] duration-[var(--duration-fast)] ease-standard active:scale-[0.98]",
        "disabled:border-neutral-200 disabled:bg-neutral-100 disabled:text-neutral-400",
        VARIANT_CLASSES[variant],
        SIZE_CLASSES[size],
        fullWidth && "w-full",
        className,
      )}
      {...props}
    >
      {loading ? (
        <Spinner size={16} />
      ) : isValidElement<{ size?: number }>(icon) ? (
        cloneElement(icon, { size: ICON_SIZE[size] })
      ) : (
        icon
      )}
      {children}
    </button>
  );
});
