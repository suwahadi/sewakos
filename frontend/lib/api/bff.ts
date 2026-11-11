import { NextResponse } from "next/server";
import { apiFetch, type ApiRequestOptions } from "./client";

export async function proxyToApi(path: string, options?: ApiRequestOptions): Promise<NextResponse> {
  const result = await apiFetch(path, options);
  if (result.ok) return NextResponse.json(result.data, { status: result.httpStatus });
  return NextResponse.json(
    { status: false, meta: result.meta, messages: result.messages },
    {
      status: result.httpStatus || 503,
      headers: result.retryAfterSeconds !== null ? { "Retry-After": String(result.retryAfterSeconds) } : undefined,
    },
  );
}
