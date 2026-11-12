"use client";

import {
  Banknote,
  Building2,
  History,
  ListChecks,
  MapPin,
  Menu,
  ReceiptText,
  Scale,
  Settings,
  Star,
  Undo2,
  UserRound,
  Users,
  Wallet,
  X,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useId, useRef, useState } from "react";
import { IconButton } from "@/components/ui/icon-button";
import { cn } from "@/lib/cn";
import { closeDialog, openDialog } from "@/lib/dialog";

interface AdminNavItem {
  href: string;
  label: string;
  icon: LucideIcon;
}

const ITEMS: AdminNavItem[] = [
  { href: "/admin/properti", label: "Moderasi properti", icon: Building2 },
  { href: "/admin/ulasan", label: "Moderasi ulasan", icon: Star },
  { href: "/admin/booking", label: "Booking", icon: ListChecks },
  { href: "/admin/tagihan", label: "Tagihan", icon: ReceiptText },
  { href: "/admin/refund", label: "Refund", icon: Undo2 },
  { href: "/admin/payout", label: "Payout", icon: Banknote },
  { href: "/admin/keuangan", label: "Keuangan", icon: Wallet },
  { href: "/admin/rekonsiliasi", label: "Rekonsiliasi", icon: Scale },
  { href: "/admin/riwayat", label: "Riwayat", icon: History },
  { href: "/admin/pengguna", label: "Pengguna", icon: Users },
  { href: "/admin/area", label: "Area", icon: MapPin },
  { href: "/admin/pengaturan", label: "Pengaturan", icon: Settings },
];

const ACCOUNT_ITEM: AdminNavItem = { href: "/akun", label: "Akun", icon: UserRound };

const DESKTOP_QUERY = "(min-width: 1024px)";

function AdminLogo() {
  return (
    <Link href="/" className="text-title-md font-bold text-brand-700">
      Admin Sewa Kos
    </Link>
  );
}

function AdminNavLink({ item, active, onNavigate }: { item: AdminNavItem; active: boolean; onNavigate?: () => void }) {
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex h-11 items-center gap-3 rounded-md px-3 text-label font-medium",
        active ? "bg-brand-50 font-semibold text-brand-800" : "text-neutral-700 hover:bg-neutral-50",
      )}
    >
      <Icon aria-hidden="true" size={20} strokeWidth={1.75} className="shrink-0" />
      <span className="truncate">{item.label}</span>
    </Link>
  );
}

function AdminNavList({ pathname, onNavigate }: { pathname: string; onNavigate?: () => void }) {
  return (
    <nav aria-label="Menu admin" className="flex-1 overflow-y-auto px-2 pb-4">
      <ul className="flex flex-col gap-1">
        {ITEMS.map((item) => (
          <li key={item.href}>
            <AdminNavLink
              item={item}
              active={pathname === item.href || pathname.startsWith(`${item.href}/`)}
              onNavigate={onNavigate}
            />
          </li>
        ))}
        <li className="mt-2 border-t border-neutral-200 pt-2">
          <AdminNavLink item={ACCOUNT_ITEM} active={false} onNavigate={onNavigate} />
        </li>
      </ul>
    </nav>
  );
}

export function AdminNav() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const drawerId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) openDialog(dialog);
    if (!open && dialog.open) closeDialog(dialog);
  }, [open]);

  useEffect(() => {
    if (typeof window.matchMedia !== "function") return;
    const desktop = window.matchMedia(DESKTOP_QUERY);
    const closeOnDesktop = (event: MediaQueryListEvent) => {
      if (event.matches) setOpen(false);
    };
    desktop.addEventListener("change", closeOnDesktop);
    return () => desktop.removeEventListener("change", closeOnDesktop);
  }, []);

  const close = () => setOpen(false);

  return (
    <>
      <header className="sticky top-0 z-[var(--z-sticky)] flex h-14 items-center gap-1 border-b border-neutral-200 bg-neutral-0 px-2 lg:hidden">
        <IconButton
          icon={<Menu aria-hidden="true" />}
          aria-label="Buka menu"
          aria-expanded={open}
          aria-controls={drawerId}
          onClick={() => setOpen(true)}
        />
        <AdminLogo />
      </header>

      <dialog
        ref={dialogRef}
        id={drawerId}
        aria-label="Menu admin"
        className="drawer bg-neutral-0 shadow-overlay"
        onClose={close}
        onCancel={close}
        onClick={(event) => {
          if (event.target === dialogRef.current) close();
        }}
      >
        <div className="flex h-full flex-col">
          <div className="flex h-14 shrink-0 items-center justify-between pl-4 pr-2">
            <AdminLogo />
            <IconButton icon={<X aria-hidden="true" />} aria-label="Tutup menu" onClick={close} />
          </div>
          <AdminNavList pathname={pathname} onNavigate={close} />
        </div>
      </dialog>

      <aside className="hidden border-r border-neutral-200 bg-neutral-0 lg:sticky lg:top-0 lg:flex lg:h-dvh lg:flex-col">
        <div className="flex h-14 shrink-0 items-center px-4">
          <AdminLogo />
        </div>
        <AdminNavList pathname={pathname} />
      </aside>
    </>
  );
}
