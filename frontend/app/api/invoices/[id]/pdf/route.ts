import { proxyBinary } from "@/lib/api/binary-proxy";

export async function GET(_request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  return proxyBinary(`/invoices/${encodeURIComponent(id)}/pdf`);
}
