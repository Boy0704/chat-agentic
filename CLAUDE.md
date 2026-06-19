# AI Agent Service — CLAUDE.md

## Project Overview

Self-hosted agentic backend (on-premise) yang bisa diinstall di server klien.
Klien (POS, Ecommerce, dll) tambah UI chatbot yang consume API service ini.
Skills ditulis klien sendiri (Python/Node) sesuai sistem mereka.

## Tech Stack

| Layer | Tech |
|---|---|
| Language | Go |
| HTTP Framework | Gin |
| DB Client | sqlx + modernc.org/sqlite (pure Go, no CGO) |
| LLM | OpenAI-compatible API (configurable — OpenAI, Ollama, Groq, dll) |
| Session Store | SQLite (`./data/agent.db`, auto-created) |
| Distribution | Docker Compose (1 service saja) |

## Keputusan Arsitektur Kritis

**Skills tidak boleh query DB kita untuk data klien.**
SQLite kita hanya untuk `sessions` table (history chat).
Data bisnis (stok, produk, transaksi) diambil via `ClientAPI` → API sistem klien.

**Tidak ada built-in skill yang touch data klien.**
Semua data skill adalah custom skill (Python/Node) yang ditulis klien sendiri.
Kita hanya sediakan: interface, loader, template, dan `ClientAPI` helper.

**OpenAI-compatible format untuk LLM.**
Klien bisa pakai model apapun — OpenAI, Ollama (lokal), Groq, Azure, dll.
Ganti `llm.base_url` dan `llm.model` di config.

## Architecture

```
Chatbot UI (milik klien)
        │  POST /api/v1/chat
        ▼
[ API Gateway — Gin ]
        │
[ Agent Core — agent.go ]
  1. Load session history dari SQLite
  2. Kirim pesan + skills sebagai tools ke LLM
  3. LLM pilih skill → Registry.Execute()
  4. ScriptSkill jalankan script klien via subprocess
  5. Script query API klien → return JSON
  6. Result dikembalikan ke LLM → final reply
        │
[ Skill Registry ]     [ Session Store ]
  custom-skills/*.py     SQLite sessions table
  auto-loaded on start   history per session_id
        │
[ ClientAPI ]
  HTTP client ke API sistem klien
  (base_url + auth dari config)
```

## Project Structure

```
agent-service/
├── cmd/server/main.go              ← bootstrap, wiring semua komponen
├── internal/
│   ├── agent/
│   │   ├── agent.go                ← LLM orchestration loop
│   │   └── event.go                ← event types untuk streaming (M2)
│   ├── api/
│   │   ├── handler.go              ← HTTP handlers
│   │   ├── middleware.go           ← Bearer token auth
│   │   └── router.go              ← route setup
│   ├── config/config.go            ← YAML + env config loader
│   ├── session/store.go            ← SQLite session store
│   ├── skill/
│   │   ├── interface.go            ← Skill interface, Manifest, Request, Result
│   │   ├── deps.go                 ← Dependencies: ClientAPI + Logger
│   │   └── registry.go            ← skill registry, ToOpenAITools()
│   └── skills/
│       ├── script_skill.go         ← wrapper untuk Python/Node/sh scripts
│       └── loader.go              ← auto-load dari custom_skills_path
├── custom-skills/                  ← skill template untuk klien
│   ├── cek_stok.manifest.json
│   └── cek_stok.py
├── scripts/test.sh                 ← manual testing script
├── config.example.yaml
├── .env.example
├── Dockerfile
└── docker-compose.yml
```

## Config

```yaml
server:
  port: 8080
  api_key: "secret"

llm:
  base_url: "https://api.openai.com/v1"   # atau Ollama: http://localhost:11434/v1
  api_key: "sk-xxx"
  model: "gpt-4o"

db:
  path: "./data/agent.db"                  # SQLite, sessions only

client_api:
  base_url: "http://client-system/api"     # URL API sistem klien
  auth_header: "Bearer xxx"
  timeout_seconds: 10

skills:
  custom_path: "./custom-skills/"          # folder .py + .manifest.json
```

## Custom Skill Contract

Script menerima params via **stdin** (JSON), return result via **stdout** (JSON):

```python
import json, sys, os
params   = json.load(sys.stdin)           # {"produk": "Indomie", ...}
base_url = os.environ["CLIENT_API_BASE_URL"]
auth     = os.environ["CLIENT_API_AUTH"]
# ... fetch data dari API klien ...
print(json.dumps({"data": [...], "summary": "teks untuk LLM"}))
```

File yang dibutuhkan per skill:
- `custom-skills/<nama>.manifest.json` — definisi parameter (OpenAI tool format)
- `custom-skills/<nama>.py` — logic skill (bisa juga `.js` atau `.sh`)

## Commands

```bash
# Development
go run cmd/server/main.go -config config.yaml

# Test
go test ./...

# Build
go build -o agent-service ./cmd/server

# Docker
docker-compose up -d

# Manual testing
API_KEY=your-key ./scripts/test.sh
```

## Environment Variables

```
SERVER_PORT=8080
SERVER_API_KEY=secret
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-xxx
LLM_MODEL=gpt-4o
DB_PATH=./data/agent.db
CLIENT_API_BASE_URL=http://client-system/api
CLIENT_API_AUTH=Bearer xxx
CUSTOM_SKILLS_PATH=./custom-skills/
```
