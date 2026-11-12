import { proxyToApi } from "@/lib/api/bff";

export async function POST(request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const body = await request.json();
  return proxyToApi(`/booking/${encodeURIComponent(id)}/review`, { method: "POST", body });
}
