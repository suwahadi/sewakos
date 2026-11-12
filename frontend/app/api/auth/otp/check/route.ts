import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";
import { setSessionToken } from "@/lib/auth/session";

interface OtpCheckResponseData {
  verified: boolean;
  token?: string;
  user?: { id: number; name: string; role: string };
}

export async function POST(request: Request) {
  const body = await request.json().catch(() => null);
  const result = await apiFetch<OtpCheckResponseData>("/auth/otp/check", { method: "POST", body, auth: false });

  if (!result.ok) {
    return NextResponse.json(
      { status: false, meta: result.meta, messages: result.messages },
      { status: result.httpStatus || 503 },
    );
  }

  const { token, ...rest } = result.data;
  if (token) {
    // Issued only for for=tenant_verification / for=login; for=reset_password never reaches
    // here with a token (API.md §2.3 — its code is consumed by /auth/password/reset instead).
    await setSessionToken(token, rest.user?.role);
  }
  return NextResponse.json(rest, { status: result.httpStatus });
}
