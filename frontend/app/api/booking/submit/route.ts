import { proxyToApi } from "@/lib/api/bff";

export async function POST(request: Request) {
  const body = await request.json().catch(() => null);
  return proxyToApi("/booking/submit", { method: "POST", body });
}
