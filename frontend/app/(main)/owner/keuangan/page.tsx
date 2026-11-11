import { AppHeader } from "@/components/features/app-header";
import { OwnerFinance } from "@/components/features/owner-finance";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import { OWNER_PAGE_SIZE, type BankAccount, type Payout } from "@/lib/owner";

export const metadata = { title: "Keuangan" };

export default async function OwnerFinancePage() {
  await requireAuthenticated("/owner/keuangan");
  const [role, bank, payouts] = await Promise.all([
    getNavRole(),
    apiFetch<{ bank_account: BankAccount | null }>("/owner/bank-account"),
    apiFetch<{ payouts: Payout[] }>("/owner/payouts", { searchParams: { limit: OWNER_PAGE_SIZE } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Keuangan" role={role} />
      {role !== "owner" ? (
        <ForbiddenState />
      ) : !bank.ok ? (
        <ErrorState title={bank.meta.message} />
      ) : (
        <OwnerFinance bankAccount={bank.data.bank_account} initialPayouts={payouts.ok ? payouts.data.payouts : []} />
      )}
    </>
  );
}
