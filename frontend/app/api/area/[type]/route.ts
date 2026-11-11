import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function GET(request: Request, context: { params: Promise<{ type: string }> }) {
  const { type } = await context.params;
  const url = new URL(request.url);
  const result = await apiFetch(`/area/${encodeURIComponent(type)}`, {
    auth: false,
    searchParams: { top: url.searchParams.get("top") ?? undefined },
  });
  if (!result.ok) {
    return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
