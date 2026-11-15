import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { OtpResendCountdown } from "./otp-input";

const past = () => Math.floor(Date.now() / 1000) - 1;

describe("OtpResendCountdown", () => {
  it("disables the resend button while a request is sending", () => {
    render(<OtpResendCountdown canResendAt={past()} onResend={vi.fn()} sending resendsLeft={2} />);

    expect(screen.getByRole("button", { name: /Kirim ulang kode/ })).toBeDisabled();
    expect(screen.getByText("Sisa 2 kali kirim ulang")).toBeInTheDocument();
  });

  it("hides the resend button once no resends are left", () => {
    render(<OtpResendCountdown canResendAt={past()} onResend={vi.fn()} resendsLeft={0} />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByText("Batas kirim ulang kode tercapai. Silakan coba lagi nanti.")).toBeInTheDocument();
  });

  it("shows the server's limit message instead of the button after a 429", () => {
    render(<OtpResendCountdown canResendAt={past()} onResend={vi.fn()} blockedMessage="Batas pengiriman kode tercapai." />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByText("Batas pengiriman kode tercapai.")).toBeInTheDocument();
  });

  it("counts down before the resend becomes available", () => {
    render(<OtpResendCountdown canResendAt={Math.floor(Date.now() / 1000) + 42} onResend={vi.fn()} resendsLeft={3} />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByText(/Kirim ulang dalam 0:4[12]/)).toBeInTheDocument();
  });
});
