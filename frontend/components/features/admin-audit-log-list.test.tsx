import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AdminAuditLogEntry } from "@/lib/admin";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { AdminAuditLogList } = await import("./admin-audit-log-list");

const entry: AdminAuditLogEntry = {
  id: 1,
  created_at: "2026-11-10T11:00:00+00:00",
  actor: { id: 3, name: "Admin Utama", role: "admin" },
  action: "user.suspend",
  entity_type: "user",
  entity_id: 9,
  changes: { suspended: [false, true] },
  ip: "203.0.113.5",
  user_agent: "Mozilla",
};

const options = { actors: [{ id: 3, name: "Admin Utama", role: "admin" }], actions: ["user.password_set", "user.suspend"] };

afterEach(() => callBffMock.mockReset());

describe("AdminAuditLogList", () => {
  it("toggles the filters on mobile and filters by actor and action", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue({ ok: true, httpStatus: 200, data: { status: true, meta: { message: "OK" }, audit_logs: [] } });
    render(<AdminAuditLogList initialEntries={[entry]} options={options} />);

    const toggle = screen.getByRole("button", { name: "Filter" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    await user.click(toggle);
    expect(screen.getByRole("button", { name: "Sembunyikan filter" })).toHaveAttribute("aria-expanded", "true");

    await user.selectOptions(screen.getByLabelText("Aktor"), "3");
    expect(callBffMock).toHaveBeenLastCalledWith("/api/admin/audit-logs?actor_id=3&page=1&limit=20");

    await user.selectOptions(screen.getByLabelText("Aksi"), "user.suspend");
    expect(callBffMock).toHaveBeenLastCalledWith("/api/admin/audit-logs?actor_id=3&action=user.suspend&page=1&limit=20");
    expect(await screen.findByText("Tidak ada aktivitas yang cocok dengan filter.")).toBeInTheDocument();
  });

  it("lists actors with their role and actions as they are recorded", () => {
    render(<AdminAuditLogList initialEntries={[entry]} options={options} />);

    expect(screen.getByRole("option", { name: "Admin Utama (Admin)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "user.password_set" })).toBeInTheDocument();
  });
});
