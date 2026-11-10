import { CircleAlert, CircleCheck, Info, TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export type InlineAlertVariant = "success" | "warning" | "danger" | "info";

const VARIANT_CLASSES: Record<InlineAlertVariant, string> = {
  success: "bg-success-bg text-success",
  warning: "bg-warning-bg text-warning",
  danger: "bg-danger-bg text-danger",
  info: "bg-info-bg text-info",
};

const VARIANT_ICON: Record<InlineAlertVariant, typeof Info> = {
  success: CircleCheck,
  warning: TriangleAlert,
  danger: CircleAlert,
  info: Info,
};

export function InlineAlert({
  variant,
  title,
  children,
  className,
}: {
  variant: InlineAlertVariant;
  title?: string;
  children: ReactNode;
  className?: string;
}) {
  const Icon = VARIANT_ICON[variant];
  return (
    <div
      role={variant === "danger" ? "alert" : "status"}
      className={cn("flex items-start gap-3 rounded-md p-3 text-neutral-900", VARIANT_CLASSES[variant], className)}
    >
      <Icon aria-hidden="true" size={20} strokeWidth={1.75} className="mt-0.5 shrink-0" />
      <div className="text-body">
        {title && <p className="font-semibold">{title}</p>}
        <div>{children}</div>
      </div>
    </div>
  );
}
