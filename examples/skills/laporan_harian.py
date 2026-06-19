#!/usr/bin/env python3
"""
Skill: laporan_harian (REST API version)
Fetches a daily sales summary report from your system's API.

Customize:
  - The API endpoint path (/reports/daily)
  - The response field names

Test locally:
  echo '{"tanggal": "2024-01-15"}' | \
    CLIENT_API_BASE_URL=http://localhost:3000 \
    CLIENT_API_AUTH="Bearer mytoken" \
    python3 examples/skills/laporan_harian.py
"""

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import date


def main():
    params = json.load(sys.stdin)
    tanggal = params.get("tanggal", str(date.today()))
    branch_id = params.get("branch_id", "")

    base_url = os.environ.get("CLIENT_API_BASE_URL", "")
    auth = os.environ.get("CLIENT_API_AUTH", "")

    query = urllib.parse.urlencode({k: v for k, v in {
        "date": tanggal,
        "branch_id": branch_id,
    }.items() if v})

    url = f"{base_url}/reports/daily?{query}"
    req = urllib.request.Request(url)
    if auth:
        req.add_header("Authorization", auth)

    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            report = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        respond({}, f"API returned error {e.code}: {e.reason}")
        return
    except urllib.error.URLError as e:
        respond({}, f"Could not connect to system API: {e.reason}")
        return

    respond(report, build_summary(tanggal, branch_id, report))


def build_summary(tanggal, branch_id, report):
    scope = f"branch {branch_id}" if branch_id else "all branches"
    total_tx = report.get("total_transactions", 0)
    revenue = report.get("total_revenue", 0)
    top = report.get("top_products", [])

    lines = [
        f"Daily report for {tanggal} ({scope}):",
        f"- Total transactions: {total_tx}",
        f"- Total revenue: {format_currency(revenue)}",
    ]

    if top:
        lines.append("- Top products:")
        for p in top[:5]:
            name = p.get("name") or p.get("nama", "Unknown")
            qty = p.get("qty_sold", 0)
            lines.append(f"  • {name}: {qty} units sold")

    return "\n".join(lines)


def format_currency(amount):
    try:
        return f"Rp {float(amount):,.0f}"
    except (TypeError, ValueError):
        return str(amount)


def respond(data, summary):
    print(json.dumps({"data": data, "summary": summary}))


if __name__ == "__main__":
    main()
