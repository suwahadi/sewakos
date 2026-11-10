import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const result = await apiFetch("/suggestion", {
    auth: false,
    searchParams: { keyword: url.searchParams.get("keyword") ?? undefined },
  });
  if (!result.ok) {
    return NextResponse.json({ status: false, meta: result.meta, messages: result.messages }, { status: result.httpStatus || 503 });
  }
  return NextResponse.json(result.data, { status: result.httpStatus });
}
