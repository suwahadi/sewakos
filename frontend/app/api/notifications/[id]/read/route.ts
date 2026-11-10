import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function PATCH(_request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const result = await apiFetch(`/notifications/${id}/read`, { method: "PATCH" });
  if (!result.ok) {
    return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
