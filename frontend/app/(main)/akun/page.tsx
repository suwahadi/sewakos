import { redirect } from "next/navigation";
import { AppHeader } from "@/components/features/app-header";
import { AkunClient } from "@/components/features/akun-client";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";

interface MeResponseData {
  user: { id: number; name: string; role: string; phone_number: string; email: string | null; is_verified: boolean };
}

export default async function AkunPage() {
  await requireAuthenticated("/akun");
  const [role, result] = await Promise.all([getNavRole(), apiFetch<MeResponseData>("/me")]);

  if (!result.ok) {
    // RequireAuth already confirmed a token exists; a failure here means it's no longer valid.
    redirect("/masuk?next=%2Fakun");
  }

  return (
    <>
      <AppHeader variant="inner" title="Akun" role={role} />
      <AkunClient user={result.data.user} />
    </>
  );
}
