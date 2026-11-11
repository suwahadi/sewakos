export interface SnapPayment {
  snap_token: string;
  redirect_url: string;
}

interface SnapCallbacks {
  onSuccess: () => void;
  onPending: () => void;
  onError: () => void;
  onClose: () => void;
}

declare global {
  interface Window {
    snap?: { pay: (token: string, callbacks: SnapCallbacks) => void };
  }
}

export const SNAP_SCRIPT_URL =
  process.env.NEXT_PUBLIC_MIDTRANS_IS_PRODUCTION === "true"
    ? "https://app.midtrans.com/snap/snap.js"
    : "https://app.sandbox.midtrans.com/snap/snap.js";

export function openSnap(payment: SnapPayment, onFinish: () => void) {
  if (!window.snap) {
    window.location.assign(payment.redirect_url);
    return;
  }
  window.snap.pay(payment.snap_token, { onSuccess: onFinish, onPending: onFinish, onError: onFinish, onClose: onFinish });
}
