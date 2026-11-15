import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { OtpChannelPicker, OtpSentNotice } from "./otp-channel-picker";

describe("OtpChannelPicker", () => {
  it("renders nothing when WhatsApp is the only channel", () => {
    const { container } = render(<OtpChannelPicker channels={["whatsapp"]} value="whatsapp" onChange={vi.fn()} />);

    expect(container).toBeEmptyDOMElement();
  });

  it("offers WhatsApp and SMS and reports the choice", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<OtpChannelPicker channels={["whatsapp", "sms"]} value="whatsapp" onChange={onChange} />);

    expect(screen.getByRole("radio", { name: "WhatsApp" })).toHaveAttribute("aria-checked", "true");
    await user.click(screen.getByRole("radio", { name: "SMS" }));

    expect(onChange).toHaveBeenCalledWith("sms");
  });
});

describe("OtpSentNotice", () => {
  it("warns that an SMS can take a while", () => {
    render(<OtpSentNotice channel="sms" phone="081234567890" />);

    expect(screen.getByText("Kode dikirim lewat SMS ke 081234567890.")).toBeInTheDocument();
    expect(screen.getByText("SMS bisa tiba hingga 1–2 menit.")).toBeInTheDocument();
  });

  it("has no delay note for WhatsApp", () => {
    render(<OtpSentNotice channel="whatsapp" phone="081234567890" />);

    expect(screen.queryByText(/SMS bisa tiba/)).not.toBeInTheDocument();
  });
});
