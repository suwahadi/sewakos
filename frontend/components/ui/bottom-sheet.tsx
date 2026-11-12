"use client";

import { X } from "lucide-react";
import { useEffect, useId, useRef, type ReactNode } from "react";
import { closeDialog, openDialog } from "@/lib/dialog";
import { IconButton } from "./icon-button";

export interface BottomSheetProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  footer?: ReactNode;
}

/** DESIGN.md §6.20 — native <dialog> (see globals.css `.sheet` for the enter/exit animation). */
export function BottomSheet({ open, onClose, title, children, footer }: BottomSheetProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) openDialog(dialog);
    if (!open && dialog.open) closeDialog(dialog);
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      className="sheet w-full rounded-t-sheet bg-neutral-0 shadow-sheet"
      aria-labelledby={titleId}
      onClose={(event) => {
        if (event.target === dialogRef.current) onClose();
      }}
      onCancel={(event) => {
        if (event.target === dialogRef.current) onClose();
      }}
      onClick={(event) => {
        if (event.target === dialogRef.current) onClose();
      }}
    >
      <div className="flex max-h-[90dvh] flex-col">
        <div className="flex justify-center pt-2">
          <span aria-hidden="true" className="h-1 w-9 rounded-full bg-neutral-300" />
        </div>
        <div className="flex items-center justify-between px-4 py-3">
          <p id={titleId} className="text-title-md font-semibold text-neutral-900">
            {title}
          </p>
          <IconButton icon={<X aria-hidden="true" />} aria-label="Tutup" onClick={onClose} />
        </div>
        <div className="flex-1 overflow-y-auto px-4">{children}</div>
        {footer && <div className="px-4 pt-4" style={{ paddingBottom: "calc(env(safe-area-inset-bottom) + 16px)" }}>{footer}</div>}
      </div>
    </dialog>
  );
}
