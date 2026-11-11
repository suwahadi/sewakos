import { proxyToApi } from "@/lib/api/bff";

export async function GET(request: Request, context: { params: Promise<{ room: string }> }) {
  const { room } = await context.params;
  const query = new URL(request.url).searchParams;
  return proxyToApi(`/booking/draft-price/${encodeURIComponent(room)}`, {
    auth: false,
    searchParams: {
      is_flash_sale: query.get("is_flash_sale") ?? undefined,
      duration: query.get("duration") ?? undefined,
    },
  });
}
