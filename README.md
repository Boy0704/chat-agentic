# Agent Service

Self-hosted AI agent backend that integrates with your existing business systems. Install it on your own server, connect it to your POS, e-commerce, or any other application via a simple API, then add a chatbot UI on top.

## How It Works

```
Your Chatbot UI
      │  POST /api/v1/chat
      ▼
Agent Service  ──────────────────────────────────────────┐
  1. Receives user message                               │
  2. Sends message + available skills to LLM             │
  3. LLM decides which skill to call                     │
  4. Skill fetches data from your system                 │
  5. LLM composes a natural language reply               │
      │                                               Your System
      ▼                                           (POS / E-commerce / ERP)
Natural language response                          REST API or Database
```

Skills are Python or Node.js scripts that you write — they know your system's schema and API, so the agent doesn't need to.

## Features

- **Bring your own LLM** — works with any OpenAI-compatible API: OpenAI, Ollama (local), Groq, Azure OpenAI, and more
- **Custom skills** — write skills in Python or Node.js, no Go required
- **Self-hosted** — runs entirely on your server, your data never leaves
- **Simple installation** — single Docker Compose command, SQLite for storage
- **Session memory** — agent remembers conversation context within a session
- **REST API** — easy to integrate with any frontend or backend

## Quick Start

**1. Clone and configure**

```bash
git clone https://github.com/yourusername/agent-service
cd agent-service
cp config.example.yaml config.yaml
```

Edit `config.yaml`:

```yaml
server:
  api_key: "your-secret-key"      # clients use this to authenticate

llm:
  base_url: "https://api.openai.com/v1"
  api_key: "sk-xxx"
  model: "gpt-4o"

client_api:
  base_url: "http://your-system/api"   # your existing system's API
  auth_header: "Bearer your-token"
```

**2. Add your first skill**

```bash
cp examples/skills/cek_stok.manifest.json custom-skills/
cp examples/skills/cek_stok.py custom-skills/
# Edit custom-skills/cek_stok.py to point to your actual API endpoint
```

**3. Run**

```bash
docker-compose build   # installs Python dependencies from custom-skills/requirements.txt
docker-compose up -d
```

Test it:

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Authorization: Bearer your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "user-1", "message": "How many units of Product X do we have?"}'
```

## Using a Local LLM (Ollama)

No API key needed, all data stays on your server:

```bash
ollama pull llama3.1:8b
```

```yaml
llm:
  base_url: "http://localhost:11434/v1"
  api_key: "ollama"
  model: "llama3.1:8b"
```

## Configuration Reference

```yaml
server:
  port: 8080                           # default: 8080
  api_key: "secret"                    # required: clients authenticate with this

llm:
  base_url: "https://api.openai.com/v1"
  api_key: "sk-xxx"
  model: "gpt-4o"
  timeout_seconds: 30

db:
  path: "./data/agent.db"              # SQLite, created automatically (sessions only)

client_api:
  base_url: "http://your-system/api"   # your existing system's base URL
  auth_header: "Bearer xxx"            # optional auth header passed to your API
  timeout_seconds: 10

skills:
  custom_path: "./custom-skills/"      # folder containing your skill files
```

All config values can be overridden with environment variables:

| Env Var | Config Key |
|---|---|
| `SERVER_API_KEY` | `server.api_key` |
| `LLM_BASE_URL` | `llm.base_url` |
| `LLM_API_KEY` | `llm.api_key` |
| `LLM_MODEL` | `llm.model` |
| `CLIENT_API_BASE_URL` | `client_api.base_url` |
| `CLIENT_API_AUTH` | `client_api.auth_header` |
| `CUSTOM_SKILLS_PATH` | `skills.custom_path` |

## Creating Skills

A skill is two files placed in your `custom-skills/` folder:

| File | Purpose |
|---|---|
| `skill_name.manifest.json` | Describes the skill to the LLM (name, description, parameters) |
| `skill_name.py` | The logic — fetches data and returns a result |

The agent reads the manifest to know when to use the skill. When called, it runs your script, passing parameters via stdin and reading the JSON result from stdout.

See [docs/SKILL_GUIDE.md](docs/SKILL_GUIDE.md) for a full guide with examples.

## API Reference

### `POST /api/v1/chat`

Send a message and get a response.

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Authorization: Bearer your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "user-123",
    "message": "What is the stock level for Indomie?",
    "context": { "branch_id": "branch-01" }
  }'
```

```json
{
  "session_id": "user-123",
  "message_id": "msg-abc",
  "reply": "Indomie Goreng has 48 units in stock at the main warehouse.",
  "skills_used": ["cek_stok"],
  "usage": { "prompt_tokens": 312, "completion_tokens": 28, "total_tokens": 340 }
}
```

`session_id` is optional — if omitted, a new session is created and the ID is returned.  
`context` is optional — passed as-is to the system prompt, useful for branch/user context.

### `GET /api/v1/sessions/:id`

Retrieve conversation history for a session.

### `DELETE /api/v1/sessions/:id`

Clear a session's history.

### `GET /api/v1/skills`

List all loaded skills.

### `GET /health`

Health check.

## Project Structure

```
agent-service/
├── custom-skills/          ← put your skill files here
│   ├── my_skill.manifest.json
│   └── my_skill.py
├── examples/skills/        ← example skills to get started
├── docs/
│   └── SKILL_GUIDE.md      ← full skill development guide
├── config.example.yaml
└── docker-compose.yml
```

## Development

```bash
# Run without Docker
cp config.example.yaml config.yaml   # fill in your values
go run cmd/server/main.go

# Run tests
go test ./...

# Build binary
go build -o agent-service ./cmd/server
```

**Requirements:** Go 1.22+, Python 3 (for Python skills)

## Contributing

Contributions are welcome. Please open an issue first to discuss what you'd like to change.

When adding skills to `examples/`, make sure they:
- Work with a real-world API pattern (not just mock data)
- Include clear comments on what to customize
- Have a matching `.manifest.json`

## License

MIT
