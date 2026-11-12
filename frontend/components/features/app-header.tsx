"use client";

import { Bell, ChevronLeft, UserRound } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { CountBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { cn } from "@/lib/cn";
import { NAV_ITEMS, type NavRole } from "./nav-items";

export interface AppHeaderProps {
  variant: "home" | "inner";
  /** variant="inner" — the current page's title (one line, truncated). Omit to show no title. */
  title?: string;
  unreadCount?: number;
  /** variant="inner" only — e.g. the favorite Heart button on a listing's detail page. */
  rightAction?: ReactNode;
  role: NavRole;
}

const ACCOUNT_HREF = "/akun";

/** DESIGN.md §6.22 — shadow only appears once the page has actually scrolled. */
export function AppHeader({ variant, title, unreadCount = 0, rightAction, role }: AppHeaderProps) {
  const router = useRouter();
  const pathname = usePathname();
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

  const isHome = variant === "home";
  const desktopLinks = NAV_ITEMS[role].filter((item) => item.href !== ACCOUNT_HREF);

  return (
    <>
      <div ref={sentinelRef} aria-hidden="true" className="h-px" />
      <header
        className={cn(
          "sticky top-0 z-[var(--z-sticky)] h-14 bg-brand-600 text-neutral-0 transition-shadow duration-[var(--duration-fast)] ease-standard [&_:focus-visible]:outline-neutral-0",
          scrolled && "shadow-raised",
        )}
      >
        <div className="mx-auto flex h-full w-full max-w-[1200px] items-center gap-3 px-4">
          {!isHome && (
            <IconButton
              icon={<ChevronLeft aria-hidden="true" />}
              aria-label="Kembali"
              variant="onBrand"
              onClick={() => router.back()}
              className="shrink-0 lg:hidden"
            />
          )}
          <Link
            href="/"
            className={cn("shrink-0 text-title-md font-bold text-neutral-0", !isHome && "hidden lg:inline")}
          >
            Sewa Kos
          </Link>
          {!isHome &&
            (title ? (
              <p className="min-w-0 flex-1 truncate text-title-md font-semibold lg:max-w-[320px] lg:flex-none lg:border-l lg:border-brand-300 lg:pl-3 lg:text-body lg:font-medium">
                {title}
              </p>
            ) : (
              <span aria-hidden="true" className="flex-1 lg:hidden" />
            ))}

          <nav aria-label="Navigasi utama" className="ml-auto hidden items-center gap-1 lg:flex">
            {desktopLinks.map((item) => {
              const active = pathname === item.href;

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
                    "rounded-md px-3 py-2 text-body-strong text-neutral-0 transition-colors duration-[var(--duration-fast)] ease-standard hover:bg-brand-700",
                    active && "bg-brand-700",
                  )}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>

          <div className={cn("flex shrink-0 items-center gap-1", isHome && "ml-auto lg:ml-0")}>
            {rightAction}
            {isHome && (
              <span className="relative inline-flex">
                <IconButton
                  icon={<Bell aria-hidden="true" />}
                  aria-label="Notifikasi"
                  variant="onBrand"
                  onClick={() => router.push("/notifikasi")}
                />
                {unreadCount > 0 && (
                  <CountBadge count={unreadCount} className="absolute -right-1 -top-1 bg-neutral-0 text-brand-700" />
                )}
              </span>
            )}
            <span className="hidden lg:inline-flex">
              {role === "guest" ? (
                <Button size="sm" onClick={() => router.push("/masuk")}>
                  Masuk
                </Button>
              ) : (
                <IconButton
                  icon={<UserRound aria-hidden="true" />}
                  aria-label="Akun"
                  variant="onBrand"
                  aria-current={pathname === ACCOUNT_HREF ? "page" : undefined}
                  onClick={() => router.push(ACCOUNT_HREF)}
                  className={pathname === ACCOUNT_HREF ? "bg-brand-700" : undefined}
                />
              )}
            </span>
          </div>
        </div>
      </header>
    </>
  );
}
