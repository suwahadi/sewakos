import { NextResponse } from "next/server";
import { proxyToApi } from "@/lib/api/bff";
import type { HttpMethod } from "@/lib/api/client";

type RouteContext = { params: Promise<{ path: string[] }> };

async function forward(request: Request, context: RouteContext) {
  const { path } = await context.params;
  if (path.some((segment) => segment === "." || segment === "..")) {
    return NextResponse.json(
      { status: false, meta: { response_code: 404, code: 404, severity: "ERROR", message: "Data tidak ditemukan." }, messages: null },
      { status: 404 },
    );
  }
  const method = request.method as HttpMethod;
  const body = method === "GET" || method === "DELETE" ? undefined : await request.json().catch(() => ({}));
  return proxyToApi(`/owner/${path.map(encodeURIComponent).join("/")}`, {
    method,
    body,
    searchParams: Object.fromEntries(new URL(request.url).searchParams),
  });
}

export { forward as DELETE, forward as GET, forward as PATCH, forward as POST, forward as PUT };
