#!/usr/bin/env python3
"""
Skill: cek_stok (direct PostgreSQL version)
Queries your system's database directly — use this when you don't have a REST API.

Requirements:
  pip install psycopg2-binary

Setup:
  Add to your config.yaml or environment:
    CLIENT_DB_DSN=postgres://user:pass@host:5432/dbname

  Pass it to Docker:
    environment:
      - CLIENT_DB_DSN=postgres://...

Test locally:
  echo '{"produk": "Indomie"}' | \
    CLIENT_DB_DSN="postgres://user:pass@localhost/posdb" \
    python3 examples/skills/cek_stok_db.py
"""

import json
import os
import sys

try:
    import psycopg2
    import psycopg2.extras
except ImportError:
    print(json.dumps({
        "data": [],
        "summary": "Missing dependency: run `pip install psycopg2-binary` on the agent server."
    }))
    sys.exit(0)


def main():
    params = json.load(sys.stdin)
    produk = params.get("produk", "").strip()
    gudang = params.get("gudang", "").strip()

    if not produk:
        respond([], "Parameter 'produk' is required.")
        return

    dsn = os.environ.get("CLIENT_DB_DSN", "")
    if not dsn:
        respond([], "CLIENT_DB_DSN environment variable is not set.")
        return

    try:
        conn = psycopg2.connect(dsn)
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        # Adjust table and column names to match your database schema
        if gudang:
            cur.execute(
                """SELECT name, sku, qty, warehouse
                   FROM products
                   WHERE name ILIKE %s AND warehouse ILIKE %s
                   ORDER BY name LIMIT 10""",
                (f"%{produk}%", f"%{gudang}%")
            )
        else:
            cur.execute(
                """SELECT name, sku, qty, warehouse
                   FROM products
                   WHERE name ILIKE %s OR sku ILIKE %s
                   ORDER BY name LIMIT 10""",
                (f"%{produk}%", f"%{produk}%")
            )

        rows = [dict(r) for r in cur.fetchall()]
        conn.close()

    except psycopg2.OperationalError as e:
        respond([], f"Could not connect to database: {e}")
        return
    except psycopg2.Error as e:
        respond([], f"Database error: {e}")
        return

    respond(rows, build_summary(produk, rows))


def build_summary(produk, rows):
    if not rows:
        return f"No products found matching '{produk}'."
    lines = [f"Found {len(rows)} product(s):"]
    for r in rows:
        line = f"- {r['name']} (SKU: {r['sku']}): {r['qty']} units"
        if r.get("warehouse"):
            line += f" at {r['warehouse']}"
        lines.append(line)
    return "\n".join(lines)


def respond(data, summary):
    print(json.dumps({"data": data, "summary": summary}))


if __name__ == "__main__":
    main()
