import { Button } from "./button";

/** DESIGN.md §6.25 — a full-width button, not auto-infinite-scroll (keeps quota use low). */
export function LoadMore({ loading, onClick }: { loading?: boolean; onClick: () => void }) {
  return (
    <Button variant="outline" fullWidth loading={loading} onClick={onClick}>
      Muat lebih banyak
    </Button>
  );
}
