import { proxyToApi } from "@/lib/api/bff";

export async function GET(request: Request) {
  const query = new URL(request.url).searchParams;
  return proxyToApi("/booking/my", {
    searchParams: { page: query.get("page") ?? undefined, limit: query.get("limit") ?? undefined },
  });
}
