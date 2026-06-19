#!/bin/sh
# AI Agent Service — install script
# Usage: ./scripts/install.sh
set -e

echo "=== AI Agent Service Setup ==="
echo ""

# Check requirements
if ! command -v docker >/dev/null 2>&1; then
    echo "Error: Docker tidak ditemukan. Install dari https://docs.docker.com/get-docker/"
    exit 1
fi

if ! docker compose version >/dev/null 2>&1 && ! command -v docker-compose >/dev/null 2>&1; then
    echo "Error: Docker Compose tidak ditemukan."
    exit 1
fi

# Copy config if not exists
if [ ! -f config.yaml ]; then
    cp config.example.yaml config.yaml
    echo "Dibuat: config.yaml"
    echo "  -> Edit config.yaml dengan API key LLM Anda sebelum lanjut."
else
    echo "Sudah ada: config.yaml (tidak diubah)"
fi

# Copy .env if not exists
if [ ! -f .env ]; then
    cp .env.example .env
    echo "Dibuat: .env"
    echo "  -> Edit .env dengan kredensial Anda."
else
    echo "Sudah ada: .env (tidak diubah)"
fi

# Create required directories
mkdir -p data custom-skills
echo "Direktori siap: data/ custom-skills/"

echo ""
echo "=== Instalasi selesai! ==="
echo ""
echo "Langkah selanjutnya:"
echo "  1. Edit config.yaml — isi llm.base_url, llm.api_key, llm.model"
echo "  2. Tambahkan custom skill ke custom-skills/ (lihat docs/SKILL_GUIDE.md)"
echo "  3. Jalankan service:"
echo ""
echo "       docker compose up -d"
echo ""
echo "  4. Test:"
echo ""
echo "       curl -s http://localhost:8080/health"
echo ""
