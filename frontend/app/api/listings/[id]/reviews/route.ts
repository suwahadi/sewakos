import { proxyToApi } from "@/lib/api/bff";

export async function GET(request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  return proxyToApi(`/listings/${encodeURIComponent(id)}/reviews`, {
    auth: false,
    searchParams: Object.fromEntries(new URL(request.url).searchParams),
  });
}
