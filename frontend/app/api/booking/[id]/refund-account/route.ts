import { proxyToApi } from "@/lib/api/bff";

export async function PUT(request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const body = await request.json().catch(() => null);
  return proxyToApi(`/booking/${encodeURIComponent(id)}/refund-account`, { method: "PUT", body });
}
