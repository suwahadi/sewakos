import { NextResponse } from "next/server";
import { apiFetch } from "@/lib/api/client";
import { clearSessionToken } from "@/lib/auth/session";

export async function POST() {
  const result = await apiFetch("/auth/logout", { method: "POST" });

  // Clear the cookie regardless of the backend call's outcome: the cookie is this browser's
  // only copy of the token, so a failed revoke call (e.g. a network blip) would otherwise
  // leave the user looking logged in on a device they just asked to log out of.
  await clearSessionToken();

  if (!result.ok) {
    return NextResponse.json(
      {
        status: true,
        meta: { response_code: 200, code: 200, severity: "OK", message: "Berhasil keluar." },
      },
      { status: 200 },
    );
  }

  return NextResponse.json(result.data, { status: result.httpStatus });
}
