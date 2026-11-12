import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/cn";

export type IconButtonSize = "md" | "sm" | "xs";
export type IconButtonVariant = "plain" | "overlay" | "onBrand";

const VISUAL_SIZE: Record<IconButtonSize, string> = {
  md: "size-11",
  sm: "size-9",
  xs: "size-8",
};

// xs keeps a 44px touch target via transparent padding even though it looks 32px.
const TOUCH_PADDING: Record<IconButtonSize, string> = {
  md: "",
  sm: "",
  xs: "before:absolute before:inset-[-6px] before:content-['']",
};

const VARIANT_CLASSES: Record<IconButtonVariant, string> = {
  plain: "border-neutral-400 bg-neutral-0 text-neutral-700 hover:bg-neutral-50 active:bg-neutral-100",
  overlay: "border-neutral-400 bg-neutral-0/92 text-neutral-700 shadow-raised hover:bg-neutral-0",
  onBrand: "border-neutral-0/70 bg-transparent text-neutral-0 hover:bg-brand-700",
};

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  size?: IconButtonSize;
  variant?: IconButtonVariant;
  icon: ReactNode;
  "aria-label": string;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { size = "md", variant = "plain", icon, disabled, className, ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      type={props.type ?? "button"}
      disabled={disabled}
      className={cn(
        "relative inline-flex items-center justify-center rounded-md border transition-[background-color] duration-[var(--duration-fast)] ease-standard active:scale-[0.98] disabled:border-neutral-200 disabled:bg-neutral-100 disabled:text-neutral-400",
        VISUAL_SIZE[size],
        TOUCH_PADDING[size],
        VARIANT_CLASSES[variant],
        className,
      )}
      {...props}
    >
      {icon}
    </button>
  );
});
