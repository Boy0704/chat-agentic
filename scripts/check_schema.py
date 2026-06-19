#!/usr/bin/env python3
"""Quick script to inspect DB schema and sample data."""
import psycopg2

conn = psycopg2.connect('host=localhost port=5432 dbname=db_alana user=boykurniawan')
cur = conn.cursor()

tables = ['item', 'item_history', 'item_history_detail', 'bookkeeping', 'bookkeeping_detail', 'inventory_location', 'location']

for t in tables:
    cur.execute("SELECT column_name, data_type FROM information_schema.columns WHERE table_schema='alana' AND table_name=%s ORDER BY ordinal_position", (t,))
    rows = cur.fetchall()
    print(f'\n=== {t} ({len(rows)} cols) ===')
    for r in rows:
        print(f'  {r[0]:40s} {r[1]}')

print("\n\n=== SAMPLE DATA ===")
for t in ['item', 'bookkeeping', 'bookkeeping_detail', 'item_history']:
    try:
        cur.execute(f"SELECT * FROM alana.{t} LIMIT 3")
        cols = [desc[0] for desc in cur.description]
        rows = cur.fetchall()
        print(f'\n--- {t} (sample {len(rows)} rows) ---')
        for r in rows:
            d = dict(zip(cols, r))
            print(f'  {d}')
    except Exception as e:
        print(f'\n--- {t}: ERROR: {e} ---')

conn.close()
