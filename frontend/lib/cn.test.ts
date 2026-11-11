import { describe, expect, it } from "vitest";
import { cn } from "./cn";

describe("cn", () => {
  it("merges conditional classes and resolves Tailwind conflicts", () => {
    expect(cn("px-2", false && "hidden", "px-4")).toBe("px-4");
  });

  it("keeps a design-token text size next to a text color", () => {
    expect(cn("text-body-strong", "bg-brand-600 text-neutral-0")).toBe("text-body-strong bg-brand-600 text-neutral-0");
    expect(cn("text-body", "text-neutral-700")).toBe("text-body text-neutral-700");
    expect(cn("text-body", "text-caption")).toBe("text-caption");
  });
});
