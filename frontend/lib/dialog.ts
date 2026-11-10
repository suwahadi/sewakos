/**
 * jsdom (used by our unit tests) doesn't implement `showModal`/`close` on `<dialog>` — feature-detect
 * so tests can still assert on content/attributes instead of crashing. All real target browsers support
 * both methods natively.
 */
export function openDialog(dialog: HTMLDialogElement) {
  if (typeof dialog.showModal === "function") {
    dialog.showModal();
  } else {
    dialog.setAttribute("open", "");
  }
}

export function closeDialog(dialog: HTMLDialogElement) {
  if (typeof dialog.close === "function") {
    dialog.close();
  } else {
    dialog.removeAttribute("open");
  }
}
