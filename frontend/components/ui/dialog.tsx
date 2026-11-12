"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { closeDialog, openDialog } from "@/lib/dialog";
import { Button } from "./button";
import { ButtonPair } from "./button-pair";

export interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

/** DESIGN.md §6.21 — centered native <dialog> (see globals.css `.dialog-center`). */
export function Dialog({ open, onClose, title, children }: DialogProps) {
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
      className="dialog-center rounded-lg bg-neutral-0 p-6 shadow-overlay"
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
      <p id={titleId} className="text-title-md font-semibold text-neutral-900">
        {title}
      </p>
      <div className="mt-2 text-body text-neutral-700">{children}</div>
    </dialog>
  );
}

export interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  destructive?: boolean;
  loading?: boolean;
}

/** DESIGN.md §6.21 — for irreversible actions: "Batal" (outline, left) + action (right), 50:50. */
export function ConfirmDialog({
  open,
  onClose,
  title,
  description,
  confirmLabel,
  onConfirm,
  destructive = true,
  loading = false,
}: ConfirmDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} title={title}>
      {description}
      <div className="mt-6">
        <ButtonPair>
          <Button variant="neutral" onClick={onClose}>
            Batal
          </Button>
          <Button variant={destructive ? "danger" : "success"} loading={loading} onClick={onConfirm}>
            {confirmLabel}
          </Button>
        </ButtonPair>
      </div>
    </Dialog>
  );
}
