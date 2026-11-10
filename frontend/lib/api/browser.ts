import { networkFailureResult, parseEnvelope, type ApiResult } from "./envelope";

export interface BrowserRequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
}

/** Client-side counterpart to `apiFetch`: calls this app's own BFF route handlers (same-origin, relative paths). */
export async function callBff<T>(path: string, options: BrowserRequestOptions = {}): Promise<ApiResult<T>> {
  const { method = "GET", body } = options;
  let response: Response;
  try {
    response = await fetch(path, {
      method,
      headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    return networkFailureResult();
  }
  return parseEnvelope<T>(response);
}
