import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AuditLogEntry } from "@/lib/owner";
import { AuditLogList } from "./audit-log-list";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const entry: AuditLogEntry = {
  id: 120,
  created_at: "2026-11-10T11:00:00+00:00",
  actor: { id: 1, name: "Pemilik Contoh", role: "owner" },
  action: "booking.approve",
  entity_type: "booking",
  entity_id: 2,
  changes: { status: ["paid", "approved"] },
};

afterEach(() => {
  callBffMock.mockReset();
});

describe("AuditLogList", () => {
  it("shows the actor, action, and changes for each entry", () => {
    render(<AuditLogList path="/api/owner/audit-logs" initialEntries={[entry]} />);
    expect(screen.getAllByText("Pemilik Contoh (owner)").length).toBeGreaterThan(0);
    expect(screen.getAllByText("booking.approve").length).toBeGreaterThan(0);
    expect(screen.getAllByText("status: paid → approved").length).toBeGreaterThan(0);
  });

  it("falls back to Sistem when there is no actor", () => {
    render(<AuditLogList path="/api/owner/audit-logs" initialEntries={[{ ...entry, actor: null }]} />);
    expect(screen.getAllByText("Sistem").length).toBeGreaterThan(0);
  });

  it("loads more entries from the given path", async () => {
    const user = userEvent.setup();
    const entries = Array.from({ length: 20 }, (_, i) => ({ ...entry, id: i + 1 }));
    callBffMock.mockResolvedValue({ ok: true, httpStatus: 200, data: { audit_logs: [{ ...entry, id: 999 }] } });
    render(<AuditLogList path="/api/owner/audit-logs" initialEntries={entries} />);

    await user.click(screen.getByRole("button", { name: "Muat lebih banyak" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/audit-logs?page=2&limit=20");
  });
});
