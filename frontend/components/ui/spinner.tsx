import { LoaderCircle } from "lucide-react";
import { cn } from "@/lib/cn";

export function Spinner({ size = 16, className }: { size?: number; className?: string }) {
  return (
    <LoaderCircle
      aria-hidden="true"
      size={size}
      strokeWidth={2}
      className={cn("animate-spin", className)}
    />
  );
}
