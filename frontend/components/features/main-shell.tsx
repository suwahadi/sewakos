"use client";

import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import { hasOwnStickyActionBar } from "./bottom-nav";

export function MainShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  return <div className={cn(!hasOwnStickyActionBar(pathname) && "pb-16 lg:pb-0")}>{children}</div>;
}
