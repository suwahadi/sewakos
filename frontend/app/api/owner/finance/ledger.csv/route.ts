import { proxyBinary } from "@/lib/api/binary-proxy";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const searchParams: Record<string, string> = {};
  for (const key of ["from", "to", "type"]) {
    const value = url.searchParams.get(key);
    if (value) searchParams[key] = value;
  }
  return proxyBinary("/owner/finance/ledger.csv", searchParams);
}
