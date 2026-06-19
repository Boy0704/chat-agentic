# Example Skills

Copy any of these to your `custom-skills/` folder and customize for your system.

| Files | Description | Approach |
|---|---|---|
| `cek_stok.py` + `cek_stok.manifest.json` | Check product stock | REST API (Python) |
| `cek_stok.js` + `cek_stok.manifest.json` | Check product stock | REST API (Node.js) |
| `cek_stok_db.py` + `cek_stok_db.manifest.json` | Check product stock | Direct PostgreSQL |
| `laporan_harian.py` + `laporan_harian.manifest.json` | Daily sales report | REST API (Python) |

## Usage

```bash
# Copy to your custom-skills folder
cp examples/skills/cek_stok.py custom-skills/
cp examples/skills/cek_stok.manifest.json custom-skills/

# Test the skill locally before deploying
echo '{"produk": "Indomie"}' | \
  CLIENT_API_BASE_URL=http://your-system/api \
  CLIENT_API_AUTH="Bearer your-token" \
  python3 custom-skills/cek_stok.py
```

For a full guide, see [docs/SKILL_GUIDE.md](../../docs/SKILL_GUIDE.md).
