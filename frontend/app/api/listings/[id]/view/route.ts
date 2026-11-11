import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function POST(_request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const result = await apiFetch(`/listings/${encodeURIComponent(id)}/view`, { method: "POST", auth: false });
  if (!result.ok) {
    return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
