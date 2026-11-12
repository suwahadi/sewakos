import { proxyToApi } from "@/lib/api/bff";

export async function GET(request: Request) {
  return proxyToApi("/favorites", { searchParams: Object.fromEntries(new URL(request.url).searchParams) });
}
