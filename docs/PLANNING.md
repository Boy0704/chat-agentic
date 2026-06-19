# AI Agent Service — Planning & Checklist

## Vision

Self-hosted agentic backend yang bisa diinstall di server klien (on-premise).
Klien connect service ini ke sistem mereka sendiri (POS, Ecommerce, dll) via `client_api` config,
lalu konsumsi chatbot API untuk tambah UI chatbot di atas aplikasi mereka.

## Keputusan Arsitektur Penting

**Skills tidak boleh menyentuh data klien langsung via DB kita.**
Approach yang dipilih: **B + C**
- **B** — Skill memanggil API sistem klien via `ClientAPI` (HTTP)
- **C** — Tidak ada built-in skill yang touch data klien; semua skill data adalah custom skill

```
Our SQLite          → sessions (history chat) SAJA, dibuat otomatis
client_api config   → URL + auth untuk API sistem klien
custom-skills/      → skill Python/Node milik klien, query sistem mereka sendiri
```

Alasan: Setiap klien punya schema DB dan API yang berbeda. Kita tidak bisa hardcode query.
Custom skill memberi klien kontrol penuh atas cara mereka fetch data.

---

## Milestone 1 — Foundation ✅

> Goal: Project bisa jalan, terima chat, panggil skill custom, return response.

### Project Setup
- [x] Init Go module
- [x] Setup folder structure
- [x] Setup Gin HTTP framework
- [x] Setup config loader (YAML + env override)
- [x] Setup structured logger (slog)
- [x] Setup Dockerfile (multi-stage, CGO_ENABLED=0)
- [x] Setup docker-compose.yml (1 service saja, SQLite)
- [x] Setup `config.example.yaml` dengan komentar

### Skill Layer
- [x] Definisi `Skill` interface (`internal/skill/interface.go`)
- [x] Definisi `Manifest`, `Request`, `Result` types
- [x] Definisi `Dependencies` — `ClientAPI` + `Logger` (`internal/skill/deps.go`)
- [x] `ClientAPI` struct dengan method `Get()` dan `Post()` ke sistem klien
- [x] Implementasi `Registry` (`internal/skill/registry.go`)
  - [x] `Register(skill Skill) error`
  - [x] `ToOpenAITools() []Tool`
  - [x] `Execute(ctx, name, params, appCtx) (Result, error)`
  - [x] `List() []Manifest`
- [x] Unit test Registry (6 test cases)

### Custom Skill System
- [x] `ScriptSkill` wrapper (`internal/skills/script_skill.go`)
  - [x] Jalankan Python/Node/sh script via subprocess
  - [x] Pass params via stdin (JSON)
  - [x] Read result dari stdout (JSON)
  - [x] Handle exit error + stderr
  - [x] Inject `CLIENT_API_BASE_URL` dan `CLIENT_API_AUTH` via env
- [x] Auto-loader dari `custom_skills_path` (`internal/skills/loader.go`)
  - [x] Scan folder untuk `*.manifest.json`
  - [x] Match dengan script pasangannya (`.py` / `.js` / `.sh`)
  - [x] Register ke Registry otomatis saat startup
- [x] Template custom skill Python (`custom-skills/cek_stok.py`)
- [x] Manifest contoh (`custom-skills/cek_stok.manifest.json`)

### Agent Core
- [x] Implementasi `Agent.Run()` (`internal/agent/agent.go`)
  - [x] Build messages dari history
  - [x] Kirim message + tools ke LLM (OpenAI-compatible)
  - [x] Handle `tool_calls` response dari LLM
  - [x] Eksekusi skill via Registry
  - [x] Kirim result kembali ke LLM
  - [x] Loop sampai LLM selesai (tidak ada tool_call lagi)
- [x] Definisi `Event` types untuk streaming (`internal/agent/event.go`)

### Session Store
- [x] Implementasi session store SQLite (`internal/session/store.go`)
  - [x] Auto-migrate tabel `sessions` saat startup
  - [x] `Get(sessionID)` — load history sebagai OpenAI messages
  - [x] `Append(sessionID, userMsg, assistantMsg)`
  - [x] `Delete(sessionID)`
  - [x] `GetHistory(sessionID)` — untuk API response

### API Layer
- [x] Auth middleware — Bearer token check
- [x] `POST /api/v1/chat` — non-streaming
- [x] `GET /health`
- [x] `GET /api/v1/skills`
- [x] `GET /api/v1/sessions/:id`
- [x] `DELETE /api/v1/sessions/:id`
- [x] Graceful shutdown (SIGTERM/SIGINT)

### Testing Milestone 1
- [ ] Kirim chat via curl, dapat response dari LLM
- [ ] Custom skill terpanggil dan return data
- [ ] History tersimpan, conversation nyambung di request berikutnya
- [ ] Unauthorized request return 401

---

## Milestone 2 — Streaming SSE ✅

> Goal: Streaming response real-time, user lihat output token per token.

### Streaming
- [x] `POST /api/v1/chat/stream` handler — SSE
- [x] `Agent.RunStream()` — kirim events ke channel
- [x] Stream token dari LLM (delta content)
- [x] Emit `skill_start` event sebelum skill dieksekusi
- [x] Emit `skill_result` event setelah skill selesai
- [x] Emit `done` event dengan metadata
- [x] Emit `error` event jika gagal
- [x] `accumulateToolCall()` — gabungkan tool call fragments dari stream chunks
- [x] Session tersimpan setelah stream selesai

### Testing Milestone 2
- [ ] Streaming response muncul token per token
- [ ] Skill events muncul di stream
- [ ] Client disconnect tidak crash server

---

## Milestone 3 — Production Ready ✅

> Goal: Bisa diinstall di server klien dengan mudah, aman, dan stable.

### Security
- [x] Rate limiting per IP atau per API key (`server.rate_limit` di config, sliding window in-memory)
- [x] Request body size limit (`server.max_body_bytes`, default 1 MB)
- [x] Timeout untuk LLM call (configurable — `llm.timeout_seconds`, applied per API call)
- [x] Timeout untuk script skill execution (`skills.timeout_seconds`, default 30s)
- [x] Sanitasi stderr dari script skill sebelum dikembalikan ke client (truncate 500 chars, strip control chars)

### Observability
- [x] Log structured untuk setiap request (method, path, status, latency) — custom slog middleware
- [x] Log setiap skill yang dieksekusi (name, duration_ms, success/error)
- [x] Log LLM token usage per request (prompt/completion/total tokens)
- [x] `GET /health` extended — cek koneksi DB + ping LLM (returns `checks` map)
- [ ] `GET /metrics` Prometheus (opsional — skip untuk sekarang)

### Configuration & Installation
- [x] Validasi `client_api.base_url` — warn jika kosong dan ada skill yang butuhnya
- [x] README instalasi lengkap (sudah ada README.md)
- [x] Script instalasi one-liner (`scripts/install.sh`)

### Docker & Deployment
- [x] Health check di docker-compose (`wget -qO- /health`, 30s interval)
- [x] Volume untuk `./data` (SQLite) dan `./custom-skills`
- [x] `.env.example` lengkap
- [ ] Verifikasi image size < 50MB (perlu build + check)

### Testing Milestone 3
- [ ] Install dari scratch via docker-compose
- [ ] Config tidak valid → exit dengan error jelas
- [ ] LLM timeout → error message proper
- [ ] Script skill timeout → error message proper

---

## Milestone 4 — Multi-App Support

> Goal: Satu service untuk beberapa klien/aplikasi dengan isolasi penuh.

### Isolasi per App
- [ ] Config multi-app
  ```yaml
  apps:
    pos:
      api_key: "sk-pos-xxx"
      skills_path: "./skills/pos/"
      client_api:
        base_url: "http://pos-api"
    ecommerce:
      api_key: "sk-ecom-xxx"
      skills_path: "./skills/ecommerce/"
      client_api:
        base_url: "http://ecom-api"
  ```
- [ ] Middleware resolve app dari API key
- [ ] Registry per-app (skill tidak bocor antar app)
- [ ] Session terisolasi per app

### Testing Milestone 4
- [ ] 2 app jalan bersamaan, skill terisolasi
- [ ] Session satu app tidak bocor ke app lain

---

## API Reference

### POST /api/v1/chat
```json
// Request
{
  "session_id": "user-123",
  "message": "Stok Indomie berapa?",
  "context": { "branch_id": "cab-001", "user_role": "kasir" }
}

// Response
{
  "session_id": "user-123",
  "message_id": "msg-abc",
  "reply": "Stok Indomie Goreng tersisa 48 pcs.",
  "skills_used": ["cek_stok"],
  "usage": { "prompt_tokens": 312, "completion_tokens": 28, "total_tokens": 340 }
}
```

### POST /api/v1/chat/stream (M2)
```
data: {"type":"skill_start","skill":"cek_stok"}
data: {"type":"token","content":"Stok Indomie"}
data: {"type":"skill_result","skill":"cek_stok","summary":"48 pcs"}
data: {"type":"done","message_id":"msg-abc","skills_used":["cek_stok"]}
```

### GET /api/v1/sessions/:id
```json
{
  "session_id": "user-123",
  "messages": [
    { "role": "user", "content": "...", "timestamp": "..." },
    { "role": "assistant", "content": "...", "timestamp": "..." }
  ]
}
```

### DELETE /api/v1/sessions/:id → `204 No Content`

### GET /api/v1/skills
```json
{ "skills": [{ "name": "cek_stok", "description": "...", "required_params": ["produk"] }] }
```

### GET /health
```json
{ "status": "ok", "version": "1.0.0" }
```

---

## Skill Development Guide

### Cara Buat Custom Skill (Python) — Cara yang Direkomendasikan

1. Buat `custom-skills/<nama>.manifest.json`:
```json
{
  "name": "nama_skill",
  "description": "Deskripsi untuk LLM",
  "parameters": {
    "type": "object",
    "properties": {
      "param1": { "type": "string", "description": "..." }
    },
    "required": ["param1"]
  }
}
```

2. Buat `custom-skills/<nama>.py`:
```python
import json, sys, os, urllib.request

params = json.load(sys.stdin)
base_url = os.environ["CLIENT_API_BASE_URL"]
auth     = os.environ["CLIENT_API_AUTH"]

# fetch dari API klien
req = urllib.request.Request(f"{base_url}/endpoint")
req.add_header("Authorization", auth)
with urllib.request.urlopen(req) as r:
    data = json.loads(r.read())

print(json.dumps({"data": data, "summary": "..."}))
```

3. Restart service → skill langsung tersedia, tidak perlu compile ulang.

### Cara Buat Custom Skill (Node.js)

```javascript
// custom-skills/nama_skill.js
const params = JSON.parse(require('fs').readFileSync('/dev/stdin', 'utf8'))
const result = { data: [], summary: '...' }
process.stdout.write(JSON.stringify(result))
```

### Environment Variables Tersedia di Script

| Var | Isi |
|---|---|
| `CLIENT_API_BASE_URL` | Base URL API sistem klien |
| `CLIENT_API_AUTH` | Auth header (contoh: `Bearer xxx`) |

---

## Progress Tracker

| Milestone | Status |
|---|---|
| M1 — Foundation | ✅ Complete |
| M2 — Streaming SSE | ✅ Complete |
| M3 — Production Ready | ✅ Complete (pending manual test) |
| M4 — Multi-App Support | 🔲 Not Started |
