import { toast } from "@/lib/toast";
import type { ApiResult } from "./envelope";
import { mapApiError } from "./errors";

type ErrorResult = Extract<ApiResult<unknown>, { ok: false }>;

export function showApiError(result: ErrorResult, navigate: (href: string) => void, retry?: () => void) {
  const error = mapApiError(result);
  switch (error.kind) {
    case "session_expired":
      toast.info(error.message);
      navigate(`/masuk?next=${encodeURIComponent(window.location.pathname + window.location.search)}`);
      return;
    case "rate_limited":
      toast.warning(error.message, { id: "rate-limit" });
      return;
    case "retryable_error":
      toast.error(error.message, retry ? { action: { label: "Coba lagi", onClick: retry } } : undefined);
      return;
    default:
      toast.error(error.message);
  }
}
