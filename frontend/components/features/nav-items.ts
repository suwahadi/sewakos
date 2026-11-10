import { Building2, ChartColumn, Heart, House, ListChecks, ReceiptText, Search, UserRound, Wallet } from "lucide-react";
import type { ComponentType } from "react";

export type NavRole = "guest" | "tenant" | "owner";

export interface NavItem {
  href: string;
  label: string;
  icon: ComponentType<{ "aria-hidden"?: boolean | "true" | "false"; size?: number; strokeWidth?: number; className?: string }>;
  /** Guests are sent to /masuk?next= instead of this page (DESIGN.md §6.23). */
  requiresAuth?: boolean;
}

const TENANT_ITEMS: NavItem[] = [
  { href: "/", label: "Beranda", icon: House },
  { href: "/cari", label: "Cari", icon: Search },
  { href: "/favorit", label: "Favorit", icon: Heart, requiresAuth: true },
  { href: "/booking", label: "Booking", icon: ReceiptText, requiresAuth: true },
  { href: "/akun", label: "Akun", icon: UserRound, requiresAuth: true },
];

const OWNER_ITEMS: NavItem[] = [
  { href: "/owner", label: "Beranda", icon: ChartColumn },
  { href: "/owner/properti", label: "Properti", icon: Building2 },
  { href: "/owner/booking", label: "Booking", icon: ListChecks },
  { href: "/owner/keuangan", label: "Keuangan", icon: Wallet },
  { href: "/akun", label: "Akun", icon: UserRound },
];

/** DESIGN.md §6.23 — item set differs by role; "guest" reuses the tenant set with auth redirects. */
export const NAV_ITEMS: Record<NavRole, NavItem[]> = {
  guest: TENANT_ITEMS,
  tenant: TENANT_ITEMS,
  owner: OWNER_ITEMS,
};
