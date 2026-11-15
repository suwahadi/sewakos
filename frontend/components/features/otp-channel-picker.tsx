"use client";

import { cn } from "@/lib/cn";
import { OTP_CHANNEL_LABEL, type OtpChannel } from "@/lib/otp-channels";

export interface OtpChannelPickerProps {
  channels: OtpChannel[];
  value: OtpChannel;
  onChange: (value: OtpChannel) => void;
}

export function OtpChannelPicker({ channels, value, onChange }: OtpChannelPickerProps) {
  if (channels.length < 2) return null;

  return (
    <div role="radiogroup" aria-label="Kirim kode lewat" className="grid grid-cols-2 gap-3">
      {channels.map((channel) => {
        const selected = channel === value;
        return (
          <button
            key={channel}
            type="button"
            role="radio"
            aria-checked={selected}
            onClick={() => onChange(channel)}
            className={cn(
              "flex h-[52px] items-center justify-center rounded-md border text-label font-medium transition-colors duration-[var(--duration-fast)] ease-standard",
              selected ? "border-2 border-brand-600 bg-brand-50 text-brand-800" : "border-neutral-300 text-neutral-700",
            )}
          >
            {OTP_CHANNEL_LABEL[channel]}
          </button>
        );
      })}
    </div>
  );
}

export function OtpSentNotice({ channel, phone, className }: { channel: OtpChannel; phone: string; className?: string }) {
  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <p className="text-body text-neutral-600">
        Kode dikirim lewat {OTP_CHANNEL_LABEL[channel]} ke {phone}.
      </p>
      {channel === "sms" && <p className="text-caption text-neutral-500">SMS bisa tiba hingga 1–2 menit.</p>}
    </div>
  );
}
