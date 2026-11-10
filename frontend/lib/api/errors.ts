import type { ApiResult } from "./envelope";

export type MappedApiError =
  | { kind: "validation"; message: string; fieldMessages: Record<string, string[]> }
  | { kind: "session_expired"; message: string }
  | { kind: "forced_logout"; message: string }
  | { kind: "not_found"; message: string }
  | { kind: "rate_limited"; message: string; retryAfterSeconds: number | null }
  | { kind: "retryable_error"; message: string }
  | { kind: "generic"; message: string };

type ErrorResult = Extract<ApiResult<unknown>, { ok: false }>;

const ACCOUNT_SUSPENDED_MESSAGE = "Akun dinonaktifkan.";

/**
 * Maps a failed API response to a UI action per DESIGN.md §8.2.
 *
 * Assumes the request was either public or already authenticated — a 401 here means
 * "the session token is invalid", so it is mapped to `session_expired`. A login attempt's
 * own 401 ("wrong credentials") is a different case (DESIGN.md §8.2: inline on the login
 * form) and must be handled by the caller before reaching this mapper, not through it.
 */
export function mapApiError(result: ErrorResult): MappedApiError {
  const { httpStatus, meta, messages, retryAfterSeconds } = result;

  if (httpStatus === 422) {
    return { kind: "validation", message: meta.message, fieldMessages: messages ?? {} };
  }
  if (httpStatus === 401) {
    return { kind: "session_expired", message: meta.message };
  }
  if (httpStatus === 403) {
    if (meta.message === ACCOUNT_SUSPENDED_MESSAGE) {
      return { kind: "forced_logout", message: meta.message };
    }
    return { kind: "generic", message: meta.message };
  }
  if (httpStatus === 404) {
    return { kind: "not_found", message: meta.message };
  }
  if (httpStatus === 429) {
    return { kind: "rate_limited", message: meta.message, retryAfterSeconds };
  }
  if (httpStatus === 500 || httpStatus === 502) {
    return { kind: "retryable_error", message: meta.message };
  }
  return { kind: "generic", message: meta.message };
}
