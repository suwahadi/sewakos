"use client";

import { ImagePlus, X } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/dialog";
import { IconButton } from "@/components/ui/icon-button";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { cn } from "@/lib/cn";
import { showApiError } from "@/lib/api/show-error";
import type { OwnerPhoto } from "@/lib/owner";
import { toast } from "@/lib/toast";
import { putWithProgress, runWithConcurrency } from "@/lib/upload";

const ACCEPTED_TYPES = ["image/jpeg", "image/png", "image/webp"];
const MAX_BYTES = 10 * 1024 * 1024;
const MAX_PHOTOS = 30;
const PARALLEL_UPLOADS = 3;
const DRAG_THRESHOLD_PX = 6;
const TOUCH_HOLD_MS = 200;
const CLICK_SUPPRESS_MS = 300;

interface PhotoGesture {
  id: number;
  pointerId: number;
  pointerType: string;
  x: number;
  y: number;
  active: boolean;
  timer?: ReturnType<typeof setTimeout>;
}

interface UploadTask {
  key: string;
  file: File;
  preview: string;
  progress: number;
  stage: "uploading" | "processing" | "failed";
  error?: string;
}

interface PresignResponse {
  upload: { url: string; headers: Record<string, string>; object_key: string };
}

export function PhotoUploader({ propertyId, initialPhotos }: { propertyId: number; initialPhotos: OwnerPhoto[] }) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [photos, setPhotos] = useState(() => [...initialPhotos].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id));
  const [tasks, setTasks] = useState<UploadTask[]>([]);
  const [selected, setSelected] = useState<OwnerPhoto | null>(null);
  const [draft, setDraft] = useState<OwnerPhoto[] | null>(null);
  const [dragId, setDragId] = useState<number | null>(null);
  const draftRef = useRef<OwnerPhoto[] | null>(null);
  const gesture = useRef<PhotoGesture | null>(null);
  const suppressClick = useRef(false);
  const taskCounter = useRef(0);
  const previews = useRef(new Set<string>());
  const view = draft ?? photos;

  useEffect(() => {
    const urls = previews.current;
    return () => urls.forEach((url) => URL.revokeObjectURL(url));
  }, []);

  function patchTask(key: string, change: Partial<UploadTask>) {
    setTasks((current) => current.map((task) => (task.key === key ? { ...task, ...change } : task)));
  }

  function dropTask(task: UploadTask) {
    URL.revokeObjectURL(task.preview);
    previews.current.delete(task.preview);
    setTasks((current) => current.filter((item) => item.key !== task.key));
  }

  async function upload(task: UploadTask) {
    patchTask(task.key, { stage: "uploading", progress: 0, error: undefined });
    const presign = await callBff<PresignResponse>("/api/owner/uploads/presign", {
      method: "POST",
      body: { content_type: task.file.type, size: task.file.size },
    });
    if (!presign.ok) return patchTask(task.key, { stage: "failed", error: presign.meta.message });
    try {
      await putWithProgress(presign.data.upload.url, presign.data.upload.headers, task.file, (progress) => patchTask(task.key, { progress }));
    } catch {
      return patchTask(task.key, { stage: "failed", error: "Unggah gagal. Periksa koneksi lalu ulangi." });
    }
    patchTask(task.key, { stage: "processing" });
    const added = await callBff<{ photo: OwnerPhoto }>(`/api/owner/properties/${propertyId}/photos`, {
      method: "POST",
      body: { object_key: presign.data.upload.object_key },
    });
    if (!added.ok) return patchTask(task.key, { stage: "failed", error: added.meta.message });
    setPhotos((current) => [...current, added.data.photo]);
    dropTask(task);
  }

  function addFiles(files: File[]) {
    let room = MAX_PHOTOS - photos.length - tasks.length;
    const accepted: UploadTask[] = [];
    for (const file of files) {
      if (!ACCEPTED_TYPES.includes(file.type)) {
        toast.error(`${file.name}: format harus JPEG, PNG, atau WebP.`);
      } else if (file.size > MAX_BYTES) {
        toast.error(`${file.name}: ukuran maksimal 10 MB.`);
      } else if (room <= 0) {
        toast.error(`${file.name}: jumlah foto sudah mencapai batas ${MAX_PHOTOS}.`);
      } else {
        room -= 1;
        taskCounter.current += 1;
        const preview = URL.createObjectURL(file);
        previews.current.add(preview);
        accepted.push({ key: `upload-${taskCounter.current}`, file, preview, progress: 0, stage: "uploading" });
      }
    }
    if (accepted.length === 0) return;
    setTasks((current) => [...current, ...accepted]);
    void runWithConcurrency(accepted, PARALLEL_UPLOADS, upload);
  }

  function previewMove(targetId: number) {
    const dragging = gesture.current?.id;
    const base = draftRef.current ?? photos;
    const from = base.findIndex((photo) => photo.id === dragging);
    const to = base.findIndex((photo) => photo.id === targetId);
    if (from < 0 || to < 0 || from === to) return;
    const next = [...base];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved!);
    draftRef.current = next;
    setDraft(next);
  }

  function activateDrag(element: HTMLElement) {
    const current = gesture.current;
    if (!current || current.active) return;
    current.active = true;
    clearTimeout(current.timer);
    try {
      element.setPointerCapture(current.pointerId);
    } catch {
      gesture.current = null;
      return;
    }
    setDragId(current.id);
    draftRef.current = photos;
    setDraft(photos);
  }

  function endDrag(commit: boolean) {
    const next = draftRef.current;
    draftRef.current = null;
    gesture.current = null;
    setDraft(null);
    setDragId(null);
    if (commit && next) void persistOrder(next);
  }

  function handlePointerDown(event: ReactPointerEvent<HTMLButtonElement>, photo: OwnerPhoto) {
    if (event.pointerType === "mouse" && event.button !== 0) return;
    const current: PhotoGesture = { id: photo.id, pointerId: event.pointerId, pointerType: event.pointerType, x: event.clientX, y: event.clientY, active: false };
    gesture.current = current;
    if (event.pointerType !== "mouse") {
      const element = event.currentTarget;
      current.timer = setTimeout(() => activateDrag(element), TOUCH_HOLD_MS);
    }
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLButtonElement>) {
    const current = gesture.current;
    if (!current || current.pointerId !== event.pointerId) return;
    if (!current.active) {
      if (Math.hypot(event.clientX - current.x, event.clientY - current.y) < DRAG_THRESHOLD_PX) return;
      if (current.pointerType !== "mouse") {
        clearTimeout(current.timer);
        gesture.current = null;
        return;
      }
      activateDrag(event.currentTarget);
    }
    const hovered = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>("[data-photo-id]");
    if (hovered?.dataset.photoId) previewMove(Number(hovered.dataset.photoId));
  }

  function handlePointerEnd(event: ReactPointerEvent<HTMLButtonElement>, cancelled: boolean) {
    const current = gesture.current;
    if (!current || current.pointerId !== event.pointerId) return;
    clearTimeout(current.timer);
    if (!current.active) {
      gesture.current = null;
      return;
    }
    suppressClick.current = true;
    setTimeout(() => {
      suppressClick.current = false;
    }, CLICK_SUPPRESS_MS);
    endDrag(!cancelled);
  }

  async function persistOrder(next: OwnerPhoto[]) {
    const previous = photos;
    setPhotos(next);
    for (const [index, photo] of next.entries()) {
      if (photo.sort_order === index + 1) continue;
      const result = await callBff(`/api/owner/photos/${photo.id}`, { method: "PATCH", body: { sort_order: index + 1 } });
      if (!result.ok) {
        setPhotos(previous);
        showApiError(result, (href) => router.push(href));
        return;
      }
    }
    setPhotos(next.map((photo, index) => ({ ...photo, sort_order: index + 1 })));
  }

  function move(photo: OwnerPhoto, offset: -1 | 1) {
    const index = photos.findIndex((item) => item.id === photo.id);
    const target = index + offset;
    if (index < 0 || target < 0 || target >= photos.length) return;
    const next = [...photos];
    [next[index], next[target]] = [next[target]!, next[index]!];
    void persistOrder(next);
  }

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-title-md font-semibold text-neutral-900">Foto</h2>
        <span className="text-caption text-neutral-600">
          {photos.length}/{MAX_PHOTOS}
        </span>
      </div>
      <div className="grid grid-cols-3 gap-2 lg:grid-cols-5">
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={photos.length + tasks.length >= MAX_PHOTOS}
          className="flex aspect-square flex-col items-center justify-center gap-1 rounded-md border-2 border-dashed border-neutral-300 text-label font-medium text-neutral-700 disabled:text-neutral-400"
        >
          <ImagePlus aria-hidden="true" size={24} strokeWidth={1.75} />
          Tambah foto
        </button>
        {view.map((photo, index) => (
          <button
            key={photo.id}
            type="button"
            data-photo-id={photo.id}
            onClick={() => {
              if (suppressClick.current) return;
              setSelected(photo);
            }}
            onPointerDown={(event) => handlePointerDown(event, photo)}
            onPointerMove={handlePointerMove}
            onPointerUp={(event) => handlePointerEnd(event, false)}
            onPointerCancel={(event) => handlePointerEnd(event, true)}
            onKeyDown={(event) => {
              if (!event.altKey || (event.key !== "ArrowLeft" && event.key !== "ArrowRight")) return;
              event.preventDefault();
              move(photo, event.key === "ArrowLeft" ? -1 : 1);
            }}
            aria-label={`Kelola foto ${index + 1}${photo.caption ? `: ${photo.caption}` : ""}`}
            aria-keyshortcuts="Alt+ArrowLeft Alt+ArrowRight"
            className={cn(
              "relative aspect-square select-none overflow-hidden rounded-md bg-neutral-100 transition-transform duration-[var(--duration-fast)] ease-standard",
              dragId === photo.id && "scale-95 opacity-80 ring-2 ring-brand-600",
            )}
          >
            <Image
              src={photo.thumbnail_url}
              alt=""
              fill
              draggable={false}
              sizes="(min-width: 1024px) 20vw, 33vw"
              className="pointer-events-none object-cover"
            />
          </button>
        ))}
        {tasks.map((task) => (
          <UploadTile key={task.key} task={task} onRetry={() => void upload(task)} onRemove={() => dropTask(task)} />
        ))}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED_TYPES.join(",")}
        multiple
        hidden
        aria-label="Pilih foto"
        onChange={(event) => {
          addFiles(Array.from(event.target.files ?? []));
          event.target.value = "";
        }}
      />
      <PhotoSheet
        photo={selected}
        onClose={() => setSelected(null)}
        onUpdated={(updated) => {
          setPhotos((current) => current.map((photo) => (photo.id === updated.id ? updated : photo)));
          setSelected(updated);
        }}
        onDeleted={(id) => {
          setPhotos((current) => current.filter((photo) => photo.id !== id));
          setSelected(null);
        }}
      />
    </section>
  );
}

function UploadTile({ task, onRetry, onRemove }: { task: UploadTask; onRetry: () => void; onRemove: () => void }) {
  const failed = task.stage === "failed";
  return (
    <div className="relative aspect-square overflow-hidden rounded-md bg-neutral-100">
      <Image src={task.preview} alt="" fill unoptimized className="object-cover" />
      <div
        role={failed ? "alert" : undefined}
        className={`absolute inset-0 flex flex-col items-center justify-center gap-1 p-1 text-center ${failed ? "bg-danger-bg" : "bg-neutral-0/80"}`}
      >
        {failed ? (
          <>
            <IconButton size="xs" aria-label="Batalkan foto ini" icon={<X aria-hidden="true" size={16} />} onClick={onRemove} className="absolute right-0 top-0" />
            <p className="line-clamp-2 text-caption text-danger">{task.error}</p>
            <Button size="sm" variant="outline" onClick={onRetry}>
              Ulangi
            </Button>
          </>
        ) : task.stage === "processing" ? (
          <p className="text-caption font-medium text-neutral-700">Memproses…</p>
        ) : (
          <ProgressRing percent={task.progress} />
        )}
      </div>
    </div>
  );
}

function ProgressRing({ percent }: { percent: number }) {
  return (
    <div className="relative size-12">
      <svg viewBox="0 0 36 36" className="size-12 -rotate-90 text-brand-600" aria-hidden="true">
        <circle cx="18" cy="18" r="16" fill="none" stroke="currentColor" strokeOpacity="0.15" strokeWidth="2" />
        <circle cx="18" cy="18" r="16" fill="none" stroke="currentColor" strokeWidth="2" pathLength={100} strokeDasharray={`${percent} 100`} />
      </svg>
      <span className="absolute inset-0 flex items-center justify-center text-caption tabular-nums text-neutral-900">{percent}%</span>
    </div>
  );
}

function PhotoSheet({
  photo,
  onClose,
  onUpdated,
  onDeleted,
}: {
  photo: OwnerPhoto | null;
  onClose: () => void;
  onUpdated: (photo: OwnerPhoto) => void;
  onDeleted: (id: number) => void;
}) {
  return (
    <BottomSheet open={photo !== null} onClose={onClose} title="Kelola foto">
      {photo && (
        <PhotoSheetBody
          key={photo.id}
          photo={photo}
          onUpdated={onUpdated}
          onDeleted={onDeleted}
        />
      )}
    </BottomSheet>
  );
}

function PhotoSheetBody({
  photo,
  onUpdated,
  onDeleted,
}: {
  photo: OwnerPhoto;
  onUpdated: (photo: OwnerPhoto) => void;
  onDeleted: (id: number) => void;
}) {
  const router = useRouter();
  const [caption, setCaption] = useState(photo.caption ?? "");
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function saveCaption() {
    setSaving(true);
    const result = await callBff(`/api/owner/photos/${photo.id}`, { method: "PATCH", body: { caption } });
    setSaving(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), saveCaption);
    toast.success(result.data.meta.message);
    onUpdated({ ...photo, caption: caption || null });
  }

  async function remove() {
    setDeleting(true);
    const result = await callBff(`/api/owner/photos/${photo.id}`, { method: "DELETE" });
    setDeleting(false);
    setConfirmDelete(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), remove);
    toast.success(result.data.meta.message);
    onDeleted(photo.id);
  }

  return (
    <div className="flex flex-col gap-4 pb-4">
      <div className="relative aspect-[4/3] overflow-hidden rounded-md bg-neutral-100">
        <Image src={photo.url} alt={photo.caption ?? ""} fill sizes="(min-width: 640px) 560px, 100vw" className="object-cover" />
      </div>
      <TextField label="Keterangan" placeholder="Tampak depan" maxLength={255} value={caption} onChange={(event) => setCaption(event.target.value)} />
      <Button variant="secondary" fullWidth loading={saving} onClick={saveCaption}>
        Simpan keterangan
      </Button>
      <Button variant="ghost" fullWidth className="text-danger hover:bg-danger-bg active:bg-danger-bg" onClick={() => setConfirmDelete(true)}>
        Hapus foto
      </Button>
      <ConfirmDialog
        open={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        title="Hapus foto?"
        description="Foto dihapus permanen dari kost ini."
        confirmLabel="Hapus"
        loading={deleting}
        onConfirm={remove}
      />
    </div>
  );
}
