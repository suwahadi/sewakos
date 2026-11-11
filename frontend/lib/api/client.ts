import { headers as requestHeaders } from "next/headers";
import { getSessionToken } from "@/lib/auth/session";
import { networkFailureResult, parseEnvelope, type ApiResult } from "./envelope";

const API_BASE_URL = process.env.API_BASE_URL;
const API_CLIENT_ID = process.env.API_CLIENT_ID;

export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export interface ApiRequestOptions {
  method?: HttpMethod;
  body?: unknown;
  /** Forward the session cookie's token as a Bearer token. Default true. */
  auth?: boolean;
  /** Explicit token to use instead of reading the session cookie (e.g. right after login). */
  token?: string;
  searchParams?: Record<string, string | number | boolean | undefined>;
}

function requireConfig(name: string, value: string | undefined): string {
  if (!value) throw new Error(`${name} is not configured`);
  return value;
}

function buildUrl(baseUrl: string, path: string, searchParams?: ApiRequestOptions["searchParams"]): string {
  const url = new URL(path.replace(/^\//, ""), `${baseUrl.replace(/\/$/, "")}/`);
  if (searchParams) {
    for (const [key, value] of Object.entries(searchParams)) {
      if (value !== undefined) url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

/** Server-side-only fetch wrapper: adds `x-api-client` and the session's Bearer token, parses the envelope. */
export async function apiFetch<T>(path: string, options: ApiRequestOptions = {}): Promise<ApiResult<T>> {
  const baseUrl = requireConfig("API_BASE_URL", API_BASE_URL);
  const clientId = requireConfig("API_CLIENT_ID", API_CLIENT_ID);
  const { method = "GET", body, auth = true, token, searchParams } = options;

  const headers: Record<string, string> = { "x-api-client": clientId };
  if (body !== undefined) headers["Content-Type"] = "application/json";

  const bearer = token ?? (auth ? await getSessionToken() : null);
  if (bearer) headers.Authorization = `Bearer ${bearer}`;
  const forwardedFor = (await requestHeaders()).get("x-forwarded-for");
  if (forwardedFor) headers["X-Forwarded-For"] = forwardedFor;

  let response: Response;
  try {
    response = await fetch(buildUrl(baseUrl, path, searchParams), {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      cache: "no-store",
    });
  } catch {
    return networkFailureResult();
  }

  return parseEnvelope<T>(response);
}
