"use client";

import { Bell, ChevronLeft } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { CountBadge } from "@/components/ui/badge";
import { IconButton } from "@/components/ui/icon-button";
import { cn } from "@/lib/cn";
import { NAV_ITEMS, type NavRole } from "./nav-items";

export interface AppHeaderProps {
  variant: "home" | "inner";
  /** Required for variant="inner" — the current page's title (one line, truncated). */
  title?: string;
  unreadCount?: number;
  /** variant="inner" only — e.g. the favorite Heart button on a listing's detail page. */
  rightAction?: ReactNode;
  role: NavRole;
}

/** DESIGN.md §6.22 — shadow only appears once the page has actually scrolled. */
export function AppHeader({ variant, title, unreadCount = 0, rightAction, role }: AppHeaderProps) {
  const router = useRouter();
  const [scrolled, setScrolled] = useState(false);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;
    const observer = new IntersectionObserver(([entry]) => setScrolled(entry ? !entry.isIntersecting : false), {
      threshold: 0,
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, []);

  const items = NAV_ITEMS[role];

  return (
    <>
      <div ref={sentinelRef} aria-hidden="true" className="h-px" />
      <header
        className={cn(
          "sticky top-0 z-[var(--z-sticky)] flex h-14 items-center bg-neutral-0 px-4 transition-shadow duration-[var(--duration-fast)] ease-standard",
          scrolled && "shadow-raised",
        )}
      >
        {variant === "home" ? (
          <div className="flex w-full items-center justify-between">
            <Link href="/" className="text-title-md font-bold text-brand-700">
              Sewa Kos
            </Link>
            <nav aria-label="Navigasi utama" className="hidden items-center gap-6 lg:flex">
              {items.map((item) => (
                <Link key={item.href} href={item.href} className="text-body-strong text-neutral-700 hover:text-brand-700">
                  {item.label}
                </Link>
              ))}
            </nav>
            <span className="relative inline-flex">
              <IconButton icon={<Bell aria-hidden="true" />} aria-label="Notifikasi" onClick={() => router.push("/notifikasi")} />
              {unreadCount > 0 && <CountBadge count={unreadCount} className="absolute -right-1 -top-1" />}
            </span>
          </div>
        ) : (
          <div className="flex w-full items-center gap-3">
            <IconButton icon={<ChevronLeft aria-hidden="true" />} aria-label="Kembali" onClick={() => router.back()} />
            <p className="min-w-0 flex-1 truncate text-title-md font-semibold text-neutral-900">{title}</p>
            {rightAction}
          </div>
        )}
      </header>
    </>
  );
}
