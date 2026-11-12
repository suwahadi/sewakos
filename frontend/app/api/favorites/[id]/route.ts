import { proxyToApi } from "@/lib/api/bff";

async function forward(method: "PUT" | "DELETE", context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  return proxyToApi(`/favorites/${encodeURIComponent(id)}`, { method });
}

export async function PUT(_request: Request, context: { params: Promise<{ id: string }> }) {
  return forward("PUT", context);
}

export async function DELETE(_request: Request, context: { params: Promise<{ id: string }> }) {
  return forward("DELETE", context);
}
