import { AppHeader } from "@/components/features/app-header";
import { FAVORITE_PAGE_SIZE, FavoriteList, type Favorite } from "@/components/features/favorite-list";
import { ErrorState, ForbiddenState } from "@/components/ui/error-state";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";

export const metadata = { title: "Favorit" };

export default async function FavoritesPage() {
  await requireAuthenticated("/favorit");
  const [role, result] = await Promise.all([
    getNavRole(),
    apiFetch<{ favorites: Favorite[] }>("/favorites", { searchParams: { limit: FAVORITE_PAGE_SIZE } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title="Favorit" role={role} />
      {role !== "tenant" ? (
        <ForbiddenState />
      ) : !result.ok ? (
        <ErrorState title={result.meta.message} />
      ) : (
        <main className="mx-auto w-full max-w-[720px] px-4 py-2">
          <FavoriteList initial={result.data.favorites} />
        </main>
      )}
    </>
  );
}
