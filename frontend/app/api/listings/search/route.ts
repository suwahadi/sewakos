import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function POST(request: Request) {
  const body = await request.json();
  const result = await apiFetch("/listings/search", { method: "POST", auth: false, body });
  if (!result.ok) {
    return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
