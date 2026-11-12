"use client";

import { useState } from "react";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { Button } from "@/components/ui/button";
import { InlineAlert } from "@/components/ui/inline-alert";
import { RatingInput } from "@/components/ui/rating-input";
import { Textarea } from "@/components/ui/textarea";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { toast } from "@/lib/toast";

export interface ReviewSheetProps {
  open: boolean;
  onClose: () => void;
  bookingId: number;
  navigate: (href: string) => void;
  onReviewed: () => void;
}

export function ReviewSheet({ open, onClose, bookingId, navigate, onReviewed }: ReviewSheetProps) {
  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [errors, setErrors] = useState<{ form?: string; rating?: string; comment?: string }>({});

  async function submit() {
    if (!rating) {
      setErrors({ rating: "Pilih rating terlebih dahulu." });
      return;
    }
    setSubmitting(true);
    setErrors({});
    const trimmed = comment.trim();
    const result = await callBff<{ review: unknown }>(`/api/booking/${bookingId}/review`, {
      method: "POST",
      body: trimmed ? { rating, comment: trimmed } : { rating },
    });
    setSubmitting(false);
    if (!result.ok) {
      if (result.httpStatus === 422) {
        setErrors({ rating: result.messages?.rating?.[0], comment: result.messages?.comment?.[0] });
        return;
      }
      if (result.httpStatus === 400) {
        setErrors({ form: result.meta.message });
        return;
      }
      showApiError(result, navigate, submit);
      return;
    }
    toast.success(result.data.meta.message);
    onClose();
    onReviewed();
  }

  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title="Tulis ulasan"
      footer={
        <Button fullWidth loading={submitting} onClick={submit}>
          Kirim ulasan
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        {errors.form && <InlineAlert variant="danger">{errors.form}</InlineAlert>}
        <RatingInput value={rating} onChange={setRating} error={errors.rating} />
        <Textarea
          label="Ulasan (opsional)"
          placeholder="Ceritakan kebersihan, kenyamanan, dan pengelolaan kost."
          maxLength={1000}
          value={comment}
          onChange={(event) => setComment(event.target.value)}
          error={errors.comment}
        />
      </div>
    </BottomSheet>
  );
}
