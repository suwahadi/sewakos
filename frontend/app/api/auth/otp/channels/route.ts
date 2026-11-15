import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function GET() {
  const result = await apiFetch("/auth/otp/channels", { auth: false });

  if (!result.ok) {
    return NextResponse.json(
      { status: false, meta: result.meta, messages: result.messages },
      { status: result.httpStatus || 503 },
    );
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
