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
        // StickyActionBar is 77px tall: a 52px button, 12px padding above and below, and a 1px top border.
        hasOwnStickyActionBar(pathname)
          ? "pb-[calc(77px+env(safe-area-inset-bottom))]"
          : "pb-[calc(4rem+env(safe-area-inset-bottom))]",
      )}
    >
      {children}
      <SiteFooter />
    </div>
  );
}
