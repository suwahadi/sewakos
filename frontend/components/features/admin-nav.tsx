"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/cn";

const ITEMS = [
  { href: "/admin/properti", label: "Moderasi properti" },
  { href: "/admin/booking", label: "Booking" },
  { href: "/admin/tagihan", label: "Tagihan" },
  { href: "/admin/refund", label: "Refund" },
  { href: "/admin/payout", label: "Payout" },
  { href: "/admin/pengguna", label: "Pengguna" },
  { href: "/admin/area", label: "Area" },
  { href: "/admin/pengaturan", label: "Pengaturan" },
];

export function AdminNav() {
  const pathname = usePathname();

  return (
    <aside className="border-b border-neutral-200 bg-neutral-0 lg:sticky lg:top-0 lg:h-dvh lg:border-b-0 lg:border-r">
      <div className="flex h-14 items-center justify-between gap-3 px-4">
        <span className="text-title-md font-bold text-brand-700">Admin Sewa Kos</span>
        <Link href="/akun" className="text-label font-semibold text-brand-700 lg:hidden">
          Akun
        </Link>
      </div>
      <nav aria-label="Menu admin">
        <ul className="flex gap-1 overflow-x-auto px-2 pb-2 [scrollbar-width:none] lg:flex-col lg:overflow-visible lg:pb-0">
          {ITEMS.map((item) => {
            const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
            return (
              <li key={item.href} className="shrink-0">
                <Link
                  href={item.href}
                  aria-current={active ? "page" : undefined}
                  className={cn(
                    "flex h-11 items-center whitespace-nowrap rounded-md px-3 text-label font-medium",
                    active ? "bg-brand-50 font-semibold text-brand-800" : "text-neutral-700 hover:bg-neutral-50",
                  )}
                >
                  {item.label}
                </Link>
              </li>
            );
          })}
          <li className="hidden lg:block">
            <Link href="/akun" className="flex h-11 items-center rounded-md px-3 text-label font-medium text-neutral-700 hover:bg-neutral-50">
              Akun
            </Link>
          </li>
        </ul>
      </nav>
    </aside>
  );
}
