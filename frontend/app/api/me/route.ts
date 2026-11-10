import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

function errorResponse(result: { meta: unknown; messages: unknown; httpStatus: number }) {
  return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
}

export async function GET() {
  const result = await apiFetch("/me");
  if (!result.ok) return errorResponse(result);
  return NextResponse.json(result.data, { status: result.httpStatus });
}

export async function PATCH(request: Request) {
  const body = await request.json().catch(() => null);
  const result = await apiFetch("/me", { method: "PATCH", body });
  if (!result.ok) return errorResponse(result);
  return NextResponse.json(result.data, { status: result.httpStatus });
}
