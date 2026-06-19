#!/usr/bin/env python3
"""Test both skills."""
import subprocess
import os

dsn = "host=localhost port=5432 dbname=db_alana user=boykurniawan"
env = os.environ.copy()
env["CLIENT_DB_DSN"] = dsn

print("=" * 60)
print("TEST 1: cek_stok_db (produk: serum)")
print("=" * 60)
p = subprocess.run(
    ["python3", "custom-skills/cek_stok_db.py"],
    input='{"produk": "serum"}',
    capture_output=True, text=True, env=env,
    cwd="/Users/boykurniawan/Development/AI/chat-agentic"
)
print("STDOUT:", p.stdout)
if p.stderr:
    print("STDERR:", p.stderr)

print("\n" + "=" * 60)
print("TEST 2: cek_bookkeeping (tanggal: 2023-11-20)")
print("=" * 60)
p2 = subprocess.run(
    ["python3", "custom-skills/cek_bookkeeping.py"],
    input='{"tanggal_awal": "2023-11-01", "tanggal_akhir": "2023-11-30"}',
    capture_output=True, text=True, env=env,
    cwd="/Users/boykurniawan/Development/AI/chat-agentic"
)
print("STDOUT:", p2.stdout)
if p2.stderr:
    print("STDERR:", p2.stderr)
