"use client";

import { Heart } from "lucide-react";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import { IconButton, type IconButtonVariant } from "@/components/ui/icon-button";
import { callBff } from "@/lib/api/browser";
import { cn } from "@/lib/cn";
import { toast } from "@/lib/toast";

export interface FavoriteButtonProps {
  propertyId: number;
  initial: boolean;
  guest: boolean;
  variant?: IconButtonVariant;
  onRemoved?: () => void;
}

export function FavoriteButton({ propertyId, initial, guest, variant = "overlay", onRemoved }: FavoriteButtonProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [active, setActive] = useState(initial);
  const [pops, setPops] = useState(0);

  async function toggle() {
    if (guest) {
      router.push(`/masuk?next=${encodeURIComponent(pathname)}`);
      return;
    }
    const next = !active;
    setActive(next);
    if (next) setPops((count) => count + 1);
    const result = await callBff(`/api/favorites/${propertyId}`, { method: next ? "PUT" : "DELETE" });
    if (!result.ok) {
      setActive(!next);
      toast.error(result.meta.message);
      return;
    }
    if (!next) onRemoved?.();
  }

  return (
    <IconButton
      variant={variant}
      aria-label={active ? "Hapus dari favorit" : "Simpan ke favorit"}
      aria-pressed={active}
      onClick={toggle}
      icon={
        <Heart
          key={pops}
          aria-hidden="true"
          size={20}
          strokeWidth={1.75}
          fill={active ? "currentColor" : "none"}
          className={cn(active && variant !== "onBrand" && "text-brand-600", active && pops > 0 && "motion-safe:animate-heart-pop")}
        />
      }
    />
  );
}
