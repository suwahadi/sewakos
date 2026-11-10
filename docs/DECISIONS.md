# DECISIONS.md — Catatan keputusan teknis kecil

> Keputusan teknis kecil (nama paket, struktur internal) yang tidak mengubah kontrak API, skema yang sudah
> di-push, atau perilaku uang (SPEC-GO.md §0.5). Keputusan yang lebih besar tetap lewat AGENT.md §3
> (berhenti dan tanyakan pemilik).

| Tanggal | Keputusan | Alasan |
|---|---|---|
| 2026-11-10 | `.githooks/pre-commit` menambah guard `[ -f backend/go.mod ]` / `[ -f frontend/package.json ]` sebelum menjalankan cek Go/npm | AGENT.md §8.2 menjalankan `npm run typecheck`/`go vet` begitu ada perubahan di `backend/`/`frontend/`, termasuk sebelum B1.1/F0.1 membuat proyeknya; tanpa guard, commit R0.3 (`frontend/.env.example`) gagal karena `package.json` belum ada. Perilaku cek untuk proyek yang sudah ada tidak berubah. |
