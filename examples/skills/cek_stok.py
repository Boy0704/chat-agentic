#!/usr/bin/env python3
"""
Skill: cek_stok (REST API version)
Checks product stock levels by calling your system's REST API.

Customize:
  - The API endpoint path (/products/stock)
  - The response field names (name, sku, qty, warehouse)

Test locally:
  echo '{"produk": "Indomie"}' | \
    CLIENT_API_BASE_URL=http://localhost:3000 \
    CLIENT_API_AUTH="Bearer mytoken" \
    python3 examples/skills/cek_stok.py
"""

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request


def main():
    params = json.load(sys.stdin)
    produk = params.get("produk", "").strip()
    gudang = params.get("gudang", "").strip()

    if not produk:
        respond([], "Parameter 'produk' is required.")
        return

    base_url = os.environ.get("CLIENT_API_BASE_URL", "")
    auth = os.environ.get("CLIENT_API_AUTH", "")

    # Build query — adjust field names to match your API
    query = urllib.parse.urlencode({k: v for k, v in {
        "search": produk,
        "warehouse": gudang,
    }.items() if v})

    url = f"{base_url}/products/stock?{query}"
    req = urllib.request.Request(url)
    if auth:
        req.add_header("Authorization", auth)

    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        respond([], f"API returned error {e.code}: {e.reason}")
        return
    except urllib.error.URLError as e:
        respond([], f"Could not connect to system API: {e.reason}")
        return

    # Adjust field names to match your API response shape
    items = body.get("data", body if isinstance(body, list) else [])
    respond(items, build_summary(produk, items))


def build_summary(produk, items):
    if not items:
        return f"No products found matching '{produk}'."
    lines = [f"Found {len(items)} product(s):"]
    for item in items:
        # Adjust field names to match your data shape
        name = item.get("name") or item.get("nama", "Unknown")
        sku = item.get("sku", "-")
        qty = item.get("qty") or item.get("stock", 0)
        warehouse = item.get("warehouse") or item.get("gudang", "")
        line = f"- {name} (SKU: {sku}): {qty} units"
        if warehouse:
            line += f" at {warehouse}"
        lines.append(line)
    return "\n".join(lines)


def respond(data, summary):
    print(json.dumps({"data": data, "summary": summary}))


if __name__ == "__main__":
    main()
