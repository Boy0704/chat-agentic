# Skill Development Guide

A skill is a unit of capability that the agent can use to fetch data from your system. The LLM decides when to call a skill based on the user's message, runs it, and uses the result to compose a natural language reply.

## How It Works

```
User: "How many units of Indomie do we have?"
           │
           ▼
LLM reads skill manifests → decides to call "cek_stok" with params: {"produk": "Indomie"}
           │
           ▼
Agent runs: python3 custom-skills/cek_stok.py
            stdin:  {"produk": "Indomie"}
            env:    CLIENT_API_BASE_URL=http://your-system/api
                    CLIENT_API_AUTH=Bearer xxx
           │
           ▼
Script fetches data from your system, prints to stdout:
{"data": [...], "summary": "Found 2 products: Indomie Goreng (48 units), Indomie Rebus (12 units)"}
           │
           ▼
LLM uses summary to compose reply:
"Indomie Goreng has 48 units and Indomie Rebus has 12 units in stock."
```

## Managing Dependencies

### Python

Add packages to `custom-skills/requirements.txt`, then rebuild the Docker image:

```
# custom-skills/requirements.txt
requests==2.32.3
psycopg2-binary==2.9.9
```

```bash
docker-compose build   # installs packages into the image
docker-compose up -d
```

The `requirements.txt` file is baked into the image at build time. You only need to rebuild when you add or change a dependency — not when you edit skill scripts.

> **Without Docker:** just `pip install` directly on the server.

### Node.js

The built-in Node.js modules (`http`, `https`, `fs`, `url`) cover most use cases and require no installation. For npm packages, install them globally on the server:

```bash
# Inside the container
docker-compose exec agent npm install -g axios

# Or on the host without Docker
npm install -g axios
```

> npm packages with `node_modules/` folders inside `custom-skills/` are not supported because the folder is volume-mounted and would override the install.

---

## File Structure

Every skill needs two files in your `custom-skills/` folder:

```
custom-skills/
├── cek_stok.manifest.json    ← describes the skill to the LLM
└── cek_stok.py               ← the actual logic
```

The filename base must match: `cek_stok.manifest.json` pairs with `cek_stok.py` (or `.js` or `.sh`).

---

## The Manifest File

The manifest tells the LLM what the skill does and what parameters it accepts. It follows the [OpenAI function calling format](https://platform.openai.com/docs/guides/function-calling).

```json
{
  "name": "cek_stok",
  "description": "Check product stock levels by name or SKU",
  "parameters": {
    "type": "object",
    "properties": {
      "produk": {
        "type": "string",
        "description": "Product name or SKU code to check"
      },
      "gudang": {
        "type": "string",
        "description": "Filter by warehouse name (optional)"
      }
    },
    "required": ["produk"]
  }
}
```

### Tips for Writing Good Descriptions

The LLM uses `description` to decide when to call your skill. Be specific:

| Bad | Good |
|---|---|
| "Get products" | "Check current stock levels for a product by name or SKU code" |
| "Sales data" | "Generate a daily sales summary report for a given date and branch" |

---

## Writing a Skill in Python

### Minimal example

```python
import json
import sys

def main():
    params = json.load(sys.stdin)
    produk = params["produk"]

    # fetch from your system here
    data = []
    summary = f"No data found for {produk}"

    print(json.dumps({"data": data, "summary": summary}))

if __name__ == "__main__":
    main()
```

### Calling your system's REST API

```python
import json
import os
import sys
import urllib.request
import urllib.parse
import urllib.error

def main():
    params = json.load(sys.stdin)
    produk = params["produk"]
    gudang = params.get("gudang", "")

    base_url = os.environ["CLIENT_API_BASE_URL"]
    auth     = os.environ["CLIENT_API_AUTH"]

    query = urllib.parse.urlencode({"search": produk, "gudang": gudang})
    url = f"{base_url}/products/stock?{query}"

    req = urllib.request.Request(url)
    req.add_header("Authorization", auth)

    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        print(json.dumps({"data": [], "summary": f"API error {e.code}: {e.reason}"}))
        return
    except urllib.error.URLError as e:
        print(json.dumps({"data": [], "summary": f"Connection error: {e.reason}"}))
        return

    items = body.get("data", [])
    print(json.dumps({
        "data": items,
        "summary": format_summary(produk, items)
    }))

def format_summary(produk, items):
    if not items:
        return f"Product '{produk}' not found."
    lines = [f"Found {len(items)} product(s):"]
    for item in items:
        line = f"- {item['name']} (SKU: {item['sku']}): {item['qty']} units"
        if item.get("warehouse"):
            line += f" at {item['warehouse']}"
        lines.append(line)
    return "\n".join(lines)

if __name__ == "__main__":
    main()
```

### Querying a database directly

If your system doesn't have an API, you can query the database directly. Install the appropriate driver first.

**PostgreSQL:**
```bash
pip install psycopg2-binary
```

```python
import json
import os
import sys
import psycopg2

def main():
    params = json.load(sys.stdin)
    produk = params["produk"]

    dsn = os.environ["CLIENT_DB_DSN"]  # add to config and pass via env
    conn = psycopg2.connect(dsn)
    cur = conn.cursor()

    cur.execute(
        "SELECT name, sku, qty, warehouse FROM products WHERE name ILIKE %s LIMIT 10",
        (f"%{produk}%",)
    )
    rows = [{"name": r[0], "sku": r[1], "qty": r[2], "warehouse": r[3]} for r in cur.fetchall()]
    conn.close()

    print(json.dumps({
        "data": rows,
        "summary": format_summary(produk, rows)
    }))

if __name__ == "__main__":
    main()
```

**MySQL:**
```bash
pip install pymysql
```

```python
import pymysql
conn = pymysql.connect(host="...", user="...", password="...", database="...")
```

---

## Writing a Skill in Node.js

```javascript
// custom-skills/cek_stok.js
const https = require('https')
const http = require('http')

async function main() {
  const params = JSON.parse(require('fs').readFileSync('/dev/stdin', 'utf8'))
  const { produk, gudang = '' } = params

  const baseUrl = process.env.CLIENT_API_BASE_URL
  const auth    = process.env.CLIENT_API_AUTH

  const url = new URL(`${baseUrl}/products/stock`)
  url.searchParams.set('search', produk)
  if (gudang) url.searchParams.set('gudang', gudang)

  try {
    const body = await get(url.toString(), auth)
    const items = body.data || []
    process.stdout.write(JSON.stringify({
      data: items,
      summary: formatSummary(produk, items)
    }))
  } catch (err) {
    process.stdout.write(JSON.stringify({
      data: [],
      summary: `Error: ${err.message}`
    }))
  }
}

function get(url, auth) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith('https') ? https : http
    const req = client.get(url, { headers: { Authorization: auth } }, res => {
      let data = ''
      res.on('data', chunk => data += chunk)
      res.on('end', () => resolve(JSON.parse(data)))
    })
    req.on('error', reject)
  })
}

function formatSummary(produk, items) {
  if (!items.length) return `Product '${produk}' not found.`
  const lines = [`Found ${items.length} product(s):`]
  for (const item of items) {
    lines.push(`- ${item.name} (SKU: ${item.sku}): ${item.qty} units`)
  }
  return lines.join('\n')
}

main()
```

---

## Writing a Skill in Shell

Useful for simple operations or when calling CLI tools:

```bash
#!/bin/sh
# custom-skills/server_status.sh

# Read params from stdin (use python3 or jq to parse JSON)
PARAMS=$(cat)
SERVER=$(echo "$PARAMS" | python3 -c "import sys,json; print(json.load(sys.stdin).get('server',''))")

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$CLIENT_API_BASE_URL/servers/$SERVER/health")

if [ "$STATUS" = "200" ]; then
    SUMMARY="Server $SERVER is online."
else
    SUMMARY="Server $SERVER appears to be down (HTTP $STATUS)."
fi

echo "{\"data\": {\"status\": \"$STATUS\"}, \"summary\": \"$SUMMARY\"}"
```

---

## Input / Output Contract

### Input (stdin)

The agent sends a JSON object with the parameters defined in your manifest:

```json
{"produk": "Indomie", "gudang": "Utama"}
```

### Output (stdout)

Your script must print a single JSON object with two fields:

```json
{
  "data": <any>,
  "summary": "Human-readable text the LLM uses to compose its reply"
}
```

- `data` — the raw result (array, object, number, etc.). The LLM may reference this for detail.
- `summary` — the most important field. Write it clearly; the LLM uses this to form its answer.

### Errors

If something goes wrong, print a result with an empty `data` and a descriptive `summary`. Do not print to stderr or exit with a non-zero code unless the skill itself crashed — the agent treats non-zero exits as hard failures.

```python
# Soft error — agent can relay this to the user
print(json.dumps({"data": [], "summary": "Could not connect to inventory API. Try again later."}))
```

---

## Environment Variables

These are available in every skill script:

| Variable | Value |
|---|---|
| `CLIENT_API_BASE_URL` | `client_api.base_url` from config |
| `CLIENT_API_AUTH` | `client_api.auth_header` from config |

You can add more by setting environment variables in your deployment or passing them through your process manager.

---

## Testing a Skill Locally

Before deploying, test your script standalone:

```bash
echo '{"produk": "Indomie"}' | CLIENT_API_BASE_URL=http://localhost:3000 \
    CLIENT_API_AUTH="Bearer mytoken" python3 custom-skills/cek_stok.py
```

Expected output:
```json
{"data": [...], "summary": "Found 2 products: ..."}
```

---

## Skill Checklist

Before deploying a new skill:

- [ ] Manifest `name` matches the script filename
- [ ] `description` clearly explains when the skill should be used
- [ ] All required parameters are listed in `required`
- [ ] Script handles missing or empty parameters gracefully
- [ ] Script always outputs valid JSON to stdout
- [ ] Tested locally with `echo '...' | python3 custom-skills/my_skill.py`
- [ ] API errors are caught and returned as a `summary` message, not a crash

---

## Examples

See the [`examples/skills/`](../examples/skills/) folder for ready-to-use templates:

| File | Description |
|---|---|
| `cek_stok.py` | Check product stock via REST API |
| `laporan_harian.py` | Daily sales report via REST API |
| `cek_stok_db.py` | Check stock by querying PostgreSQL directly |
| `cek_stok.js` | Check stock — Node.js version |
