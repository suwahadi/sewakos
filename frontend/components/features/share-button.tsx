"use client";

import { Share2 } from "lucide-react";
import { IconButton, type IconButtonVariant } from "@/components/ui/icon-button";
import { toast } from "@/lib/toast";

export interface ShareButtonProps {
  title: string;
  variant?: IconButtonVariant;
}

export function ShareButton({ title, variant = "onBrand" }: ShareButtonProps) {
  async function share() {
    const url = window.location.href;
    if (typeof navigator.share === "function") {
      try {
        await navigator.share({ title, url });
        return;
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
      }
    }
    try {
      await navigator.clipboard.writeText(url);
      toast.success("Tautan disalin");
    } catch {
      toast.error("Tautan tidak bisa disalin");
    }
  }

  return (
    <IconButton
      variant={variant}
      aria-label="Bagikan"
      onClick={share}
      icon={<Share2 aria-hidden="true" size={20} strokeWidth={1.75} />}
    />
  );
}
