import type { NextConfig } from "next";

function storagePattern() {
  const base = process.env.STORAGE_PUBLIC_BASE_URL;
  if (!base) return [];
  const url = new URL(base);
  return [
    {
      protocol: url.protocol.replace(":", "") as "http" | "https",
      hostname: url.hostname,
      port: url.port || undefined,
      pathname: `${url.pathname === "/" ? "" : url.pathname}/**`,
    },
  ];
}

const nextConfig: NextConfig = {
  images: {
    remotePatterns: storagePattern(),
  },
};

export default nextConfig;
