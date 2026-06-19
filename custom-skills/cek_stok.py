#!/usr/bin/env python3
"""
Custom Skill: cek_stok
Template untuk mengecek stok produk dari sistem klien.

Input  (stdin) : JSON params dari agent  {"produk": "...", "gudang": "..."}
Output (stdout): JSON result             {"data": [...], "summary": "..."}

Env vars yang tersedia:
  CLIENT_API_BASE_URL  - base URL API sistem klien
  CLIENT_API_AUTH      - auth header (contoh: "Bearer xxx")
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
        output([], "Parameter 'produk' wajib diisi.")
        return

    base_url = os.environ.get("CLIENT_API_BASE_URL", "")
    auth = os.environ.get("CLIENT_API_AUTH", "")

    # ── Pilih salah satu opsi di bawah, sesuai sistem klien ──────

    # OPSI A: Panggil REST API klien
    # Uncomment dan sesuaikan endpoint dengan API sistem klien
    #
    # query = urllib.parse.urlencode({"search": produk, "gudang": gudang})
    # url = f"{base_url}/products/stock?{query}"
    # req = urllib.request.Request(url)
    # if auth:
    #     req.add_header("Authorization", auth)
    # try:
    #     with urllib.request.urlopen(req, timeout=10) as resp:
    #         data = json.loads(resp.read())["data"]
    #     output(data, format_summary(produk, data))
    # except urllib.error.URLError as e:
    #     output([], f"Gagal koneksi ke API klien: {e}")

    # OPSI B: Query langsung ke database klien
    # Install driver dulu: pip install psycopg2-binary (PostgreSQL) atau pymysql (MySQL)
    #
    # import psycopg2
    # dsn = os.environ.get("CLIENT_DB_DSN", "")
    # conn = psycopg2.connect(dsn)
    # cur = conn.cursor()
    # query = "SELECT nama, sku, qty, gudang FROM products WHERE nama ILIKE %s"
    # args = [f"%{produk}%"]
    # if gudang:
    #     query += " AND gudang ILIKE %s"
    #     args.append(f"%{gudang}%")
    # cur.execute(query, args)
    # data = [{"nama": r[0], "sku": r[1], "qty": r[2], "gudang": r[3]} for r in cur.fetchall()]
    # conn.close()
    # output(data, format_summary(produk, data))

    # ── PLACEHOLDER — ganti dengan opsi A atau B di atas ─────────
    output([], f"Skill belum dikonfigurasi. Edit file custom-skills/cek_stok.py")


def format_summary(produk, data):
    if not data:
        return f"Produk '{produk}' tidak ditemukan."
    lines = [f"Ditemukan {len(data)} produk:"]
    for item in data:
        line = f"- {item['nama']} (SKU: {item['sku']}): {item['qty']} unit"
        if item.get("gudang"):
            line += f" di gudang {item['gudang']}"
        lines.append(line)
    return "\n".join(lines)


def output(data, summary):
    print(json.dumps({"data": data, "summary": summary}))


if __name__ == "__main__":
    main()
