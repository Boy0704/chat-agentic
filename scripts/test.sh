#!/bin/sh
# Manual testing script untuk semua endpoint
# Usage: ./scripts/test.sh
# Atau:  API_KEY=xxx BASE_URL=http://localhost:8080 ./scripts/test.sh

BASE_URL=${BASE_URL:-http://localhost:8080}
API_KEY=${API_KEY:-your-secret-api-key}
SESSION_ID="test-session-$(date +%s)"
AUTH="Authorization: Bearer $API_KEY"
CT="Content-Type: application/json"

echo "======================================"
echo " Agent Service — Manual Test"
echo " BASE_URL : $BASE_URL"
echo " SESSION  : $SESSION_ID"
echo "======================================"

# ── 1. Health check ──────────────────────
echo ""
echo "[1] GET /health"
curl -s "$BASE_URL/health" | python3 -m json.tool
echo ""

# ── 2. List skills yang dimuat ───────────
echo "[2] GET /api/v1/skills"
curl -s -H "$AUTH" "$BASE_URL/api/v1/skills" | python3 -m json.tool
echo ""

# ── 3. Chat — pertanyaan umum (tanpa skill) ──
echo "[3] POST /api/v1/chat — pertanyaan umum"
curl -s -X POST "$BASE_URL/api/v1/chat" \
    -H "$AUTH" -H "$CT" \
    -d "{\"session_id\":\"$SESSION_ID\",\"message\":\"Halo, kamu bisa bantu apa?\"}" \
    | python3 -m json.tool
echo ""

# ── 4. Chat — trigger skill ──────────────
echo "[4] POST /api/v1/chat — trigger cek_stok skill"
curl -s -X POST "$BASE_URL/api/v1/chat" \
    -H "$AUTH" -H "$CT" \
    -d "{\"session_id\":\"$SESSION_ID\",\"message\":\"Cek stok produk Indomie\"}" \
    | python3 -m json.tool
echo ""

# ── 5. Chat — follow-up (test memory/session) ──
echo "[5] POST /api/v1/chat — follow-up message (test session memory)"
curl -s -X POST "$BASE_URL/api/v1/chat" \
    -H "$AUTH" -H "$CT" \
    -d "{\"session_id\":\"$SESSION_ID\",\"message\":\"Tadi kamu bilang apa?\"}" \
    | python3 -m json.tool
echo ""

# ── 6. Chat — dengan context dari app klien ──
echo "[6] POST /api/v1/chat — dengan app context"
curl -s -X POST "$BASE_URL/api/v1/chat" \
    -H "$AUTH" -H "$CT" \
    -d "{
        \"session_id\": \"ctx-session\",
        \"message\": \"Stok apa saja yang tersedia?\",
        \"context\": {\"branch_id\": \"cab-001\", \"user_role\": \"kasir\"}
    }" \
    | python3 -m json.tool
echo ""

# ── 7. Get session history ───────────────
echo "[7] GET /api/v1/sessions/$SESSION_ID"
curl -s -H "$AUTH" "$BASE_URL/api/v1/sessions/$SESSION_ID" | python3 -m json.tool
echo ""

# ── 8. Delete session ────────────────────
echo "[8] DELETE /api/v1/sessions/$SESSION_ID"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "$AUTH" \
    "$BASE_URL/api/v1/sessions/$SESSION_ID")
echo "HTTP Status: $STATUS (expected 204)"
echo ""

# ── 9. Streaming SSE ─────────────────────
echo "[9] POST /api/v1/chat/stream — streaming response"
echo "(Ctrl+C untuk stop jika LLM lambat)"
curl -sN -X POST "$BASE_URL/api/v1/chat/stream" \
    -H "$AUTH" -H "$CT" \
    -d "{\"session_id\":\"stream-test\",\"message\":\"Halo, ceritakan sedikit tentang dirimu\"}" \
    --max-time 30
echo ""
echo ""

# ── 10. Unauthorized request ─────────────
echo "[10] Request tanpa auth (harus 401)"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/chat/stream" \
    -H "$CT" -d "{\"message\":\"test\"}")
echo "HTTP Status: $STATUS (expected 401)"
echo ""

echo "======================================"
echo " Test selesai"
echo "======================================"
