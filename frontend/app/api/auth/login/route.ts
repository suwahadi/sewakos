import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";
import { setSessionToken } from "@/lib/auth/session";

interface LoginResponseData {
  token: string;
  user: { id: number; name: string; role: string; is_verified: boolean };
}

export async function POST(request: Request) {
  const body = await request.json().catch(() => null);

  const result = await apiFetch<LoginResponseData>("/auth/login", {
    method: "POST",
    body,
    auth: false,
  });

  if (!result.ok) {
    return NextResponse.json(
      { status: false, meta: result.meta, messages: result.messages },
      { status: result.httpStatus || 503 },
    );
  }

  // The token stays server-side in an httpOnly cookie; it is never sent to the browser's JS.
  await setSessionToken(result.data.token, result.data.user.role);
  return NextResponse.json(
    { status: true, meta: result.data.meta, user: result.data.user },
    { status: result.httpStatus },
  );
}
