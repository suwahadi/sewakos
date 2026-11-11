import { headers as requestHeaders } from "next/headers";
import { NextResponse } from "next/server";
import { getSessionToken } from "@/lib/auth/session";
import { buildUrl, requireConfig, type ApiRequestOptions } from "./client";
import { parseEnvelope } from "./envelope";

const API_BASE_URL = process.env.API_BASE_URL;
const API_CLIENT_ID = process.env.API_CLIENT_ID;

/** Streams a non-JSON API response (PDF, CSV) through the server so the session token never reaches the browser. */
export async function proxyBinary(path: string, searchParams?: ApiRequestOptions["searchParams"]): Promise<NextResponse> {
  const baseUrl = requireConfig("API_BASE_URL", API_BASE_URL);
  const clientId = requireConfig("API_CLIENT_ID", API_CLIENT_ID);

  const headers: Record<string, string> = { "x-api-client": clientId };
  const token = await getSessionToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const forwardedFor = (await requestHeaders()).get("x-forwarded-for");
  if (forwardedFor) headers["X-Forwarded-For"] = forwardedFor;

  let response: Response;
  try {
    response = await fetch(buildUrl(baseUrl, path, searchParams), { headers, cache: "no-store" });
  } catch {
    return NextResponse.json(
      { status: false, meta: { response_code: 0, code: 0, severity: "ERROR", message: "Tidak ada koneksi internet. Periksa jaringan lalu coba lagi." }, messages: null },
      { status: 503 },
    );
  }

  if ((response.headers.get("Content-Type") ?? "").includes("application/json")) {
    const result = await parseEnvelope(response);
    if (!result.ok) {
      return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
    }
    return NextResponse.json(result.data, { status: result.httpStatus });
  }

  return new NextResponse(response.body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "application/octet-stream",
      "Content-Disposition": response.headers.get("Content-Disposition") ?? "attachment",
    },
  });
}
