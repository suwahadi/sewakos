"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { MouseEvent } from "react";
import { cn } from "@/lib/cn";
import { NAV_ITEMS, type NavRole } from "./nav-items";

export interface BottomNavProps {
  role: NavRole;
}

/** DESIGN.md §6.23 — hidden at ≥1024px (AppHeader's desktop nav takes over) and on pages with a StickyActionBar. */
export function BottomNav({ role }: BottomNavProps) {
  const pathname = usePathname();
  const router = useRouter();
  const items = NAV_ITEMS[role];

  return (
    <nav
      aria-label="Navigasi utama"
      className="fixed inset-x-0 bottom-0 z-[var(--z-sticky)] flex h-16 border-t border-neutral-200 bg-neutral-0 lg:hidden"
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      {items.map((item) => {
        const active = pathname === item.href;
        const Icon = item.icon;

        function handleClick(event: MouseEvent<HTMLAnchorElement>) {
          if (item.requiresAuth && role === "guest") {
            event.preventDefault();
            router.push(`/masuk?next=${encodeURIComponent(item.href)}`);
          }
        }

        return (
          <Link
            key={item.href}
            href={item.href}
            onClick={handleClick}
            aria-current={active ? "page" : undefined}
            className={cn(
              "relative flex flex-1 flex-col items-center justify-center gap-1",
              active ? "text-brand-700" : "text-neutral-500",
            )}
          >
            {active && <span aria-hidden="true" className="absolute top-0 h-[3px] w-6 rounded-full bg-brand-600" />}
            <Icon aria-hidden="true" size={24} strokeWidth={1.75} />
            <span className="text-label font-medium">{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
