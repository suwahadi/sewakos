import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/cn";
import { Button } from "./button";

export interface EmptyStateProps {
  icon: LucideIcon;
  iconClassName?: string;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
  className?: string;
}

export function EmptyState({ icon: Icon, iconClassName, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center gap-3 px-4 py-8 text-center md:py-12", className)}>
      <span className="flex size-[72px] items-center justify-center rounded-full bg-neutral-50">
        <Icon aria-hidden="true" size={40} strokeWidth={1.5} className={cn("text-neutral-400", iconClassName)} />
      </span>
      <p className="text-title-md font-semibold text-neutral-900">{title}</p>
      {description && <p className="line-clamp-2 max-w-prose text-body text-neutral-600">{description}</p>}
      {action && (
        <Button onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  );
}
