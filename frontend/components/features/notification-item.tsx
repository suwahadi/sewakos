import { Mail, ReceiptText, Wallet, Zap, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/cn";
import { formatRelativeTime } from "@/lib/format";

export interface Notification {
  id: number;
  category: "booking" | "payment" | "promo" | "chat";
  title: string;
  body: string;
  link: string | null;
  read_at: string | null;
  created_at: string;
}

const CATEGORY_ICONS: Record<Notification["category"], LucideIcon> = {
  booking: ReceiptText,
  payment: Wallet,
  promo: Zap,
  chat: Mail,
};

export function NotificationItem({ notification, onClick }: { notification: Notification; onClick: () => void }) {
  const Icon = CATEGORY_ICONS[notification.category];
  const unread = notification.read_at === null;

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex min-h-[72px] w-full items-center gap-3 p-4 text-left transition-colors duration-[var(--duration-fast)] ease-standard",
        unread ? "bg-brand-50" : "bg-transparent",
      )}
    >
      <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-neutral-100">
        <Icon aria-hidden="true" size={20} className="text-neutral-700" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-title-sm font-semibold text-neutral-900">{notification.title}</span>
        <span className="line-clamp-3 block text-body text-neutral-700">{notification.body}</span>
        <span className="block text-caption text-neutral-500">{formatRelativeTime(notification.created_at)}</span>
      </span>
      {unread && <span aria-hidden="true" className="size-2 shrink-0 rounded-full bg-brand-600" />}
    </button>
  );
}
