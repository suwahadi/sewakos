import { proxyToApi } from "@/lib/api/bff";

export async function POST(_request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  return proxyToApi(`/booking/${encodeURIComponent(id)}/cancel`, { method: "POST" });
}
