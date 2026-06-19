#!/usr/bin/env node
/**
 * Skill: cek_stok (Node.js version)
 * Checks product stock levels by calling your system's REST API.
 *
 * Test locally:
 *   echo '{"produk":"Indomie"}' | \
 *     CLIENT_API_BASE_URL=http://localhost:3000 \
 *     CLIENT_API_AUTH="Bearer mytoken" \
 *     node examples/skills/cek_stok.js
 */

const http = require('http')
const https = require('https')
const { URL, URLSearchParams } = require('url')

async function main() {
  const params = JSON.parse(require('fs').readFileSync('/dev/stdin', 'utf8'))
  const produk = (params.produk || '').trim()
  const gudang = (params.gudang || '').trim()

  if (!produk) {
    respond([], "Parameter 'produk' is required.")
    return
  }

  const baseUrl = process.env.CLIENT_API_BASE_URL || ''
  const auth    = process.env.CLIENT_API_AUTH || ''

  const url = new URL(`${baseUrl}/products/stock`)
  url.searchParams.set('search', produk)
  if (gudang) url.searchParams.set('warehouse', gudang)

  try {
    const body = await get(url.toString(), auth)
    const items = body.data || (Array.isArray(body) ? body : [])
    respond(items, buildSummary(produk, items))
  } catch (err) {
    respond([], `Error: ${err.message}`)
  }
}

function get(url, auth) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith('https') ? https : http
    const options = { headers: {} }
    if (auth) options.headers['Authorization'] = auth

    client.get(url, options, res => {
      let data = ''
      res.on('data', chunk => { data += chunk })
      res.on('end', () => {
        if (res.statusCode >= 400) {
          reject(new Error(`HTTP ${res.statusCode}: ${data}`))
          return
        }
        try {
          resolve(JSON.parse(data))
        } catch {
          reject(new Error('Invalid JSON response from API'))
        }
      })
    }).on('error', reject)
  })
}

function buildSummary(produk, items) {
  if (!items.length) return `No products found matching '${produk}'.`
  const lines = [`Found ${items.length} product(s):`]
  for (const item of items) {
    const name      = item.name || item.nama || 'Unknown'
    const sku       = item.sku || '-'
    const qty       = item.qty ?? item.stock ?? 0
    const warehouse = item.warehouse || item.gudang || ''
    let line = `- ${name} (SKU: ${sku}): ${qty} units`
    if (warehouse) line += ` at ${warehouse}`
    lines.push(line)
  }
  return lines.join('\n')
}

function respond(data, summary) {
  process.stdout.write(JSON.stringify({ data, summary }))
}

main().catch(err => {
  process.stdout.write(JSON.stringify({ data: [], summary: `Unexpected error: ${err.message}` }))
})
