import { CircleAlert, RefreshCw } from "lucide-react";
import { cn } from "@/lib/cn";
import { Button } from "./button";

export interface ErrorStateProps {
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void; icon?: boolean };
  className?: string;
}

/** DESIGN.md §6.19 — same anatomy as EmptyState with a danger CircleAlert icon. */
export function ErrorState({ title, description, action, className }: ErrorStateProps) {
  return (
    <div className={cn("flex flex-col items-center gap-3 px-4 py-8 text-center md:py-12", className)}>
      <span className="flex size-[72px] items-center justify-center rounded-full bg-neutral-50">
        <CircleAlert aria-hidden="true" size={40} strokeWidth={1.5} className="text-danger" />
      </span>
      <p className="text-title-md font-semibold text-neutral-900">{title}</p>
      {description && <p className="line-clamp-2 max-w-prose text-body text-neutral-600">{description}</p>}
      {action && (
        <Button
          variant="neutral"
          onClick={action.onClick}
          icon={action.icon ? <RefreshCw aria-hidden="true" /> : undefined}
        >
          {action.label}
        </Button>
      )}
    </div>
  );
}

/** Convenience preset for API.md 403/404 errors reused across guarded routes (F0.4). */
export function NotFoundState({ onGoHome }: { onGoHome: () => void }) {
  return (
    <ErrorState
      title="Halaman tidak ditemukan"
      action={{ label: "Kembali ke beranda", onClick: onGoHome }}
    />
  );
}

export function ForbiddenState() {
  return <ErrorState title="Halaman ini tidak tersedia untuk akunmu." />;
}
