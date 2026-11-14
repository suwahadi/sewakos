"use client";

import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import { hasOwnStickyActionBar } from "./bottom-nav";
import { SiteFooter } from "./site-footer";

export function MainShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  return (
    <div
      className={cn(
        "flex min-h-dvh flex-col lg:pb-0",
        // StickyActionBar is 77px tall on mobile (52px button, 12px padding above and below, 1px top border);
        // on desktop it floats 16px above the bottom edge, so 112px keeps the footer clear of it.
        hasOwnStickyActionBar(pathname)
          ? "pb-[calc(77px+env(safe-area-inset-bottom))] lg:pb-28"
          : "pb-[calc(4rem+env(safe-area-inset-bottom))]",
      )}
    >
      {children}
      <SiteFooter />
    </div>
  );
}
