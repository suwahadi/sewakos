import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function POST(request: Request) {
  const body = await request.json().catch(() => null);
  const result = await apiFetch("/auth/register", { method: "POST", body, auth: false });

  if (!result.ok) {
    return NextResponse.json(
      { status: false, meta: result.meta, messages: result.messages },
      { status: result.httpStatus || 503 },
    );
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
