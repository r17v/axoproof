# Module 4: Live Action & Universal Persistence (`live-persistence`)

Passing `tsc`, `npm run build`, or `go build` proves syntax only; `axoproof` requires live behavioral proof.

## Wiring `.axoproof/claims.json` Across Any Stack

1. **Auto-Boot Support (`start_command`):** Set `"base_url"`, `"health_path"`, and `"start_command"` (e.g., `"npm run start"` or `"go run ./cmd/server"`). If the server is not already running, `axoproof verify` starts it automatically, runs all probes, and terminates the process tree cleanly.
2. **PostgreSQL / Supabase / MySQL / API (`read_back_path`):** Set `"read_back_path"` (plus `"read_headers"` if auth-protected) and `"expect_read_substring"` (a unique SKU/reference in `"payload_json"`). `axoproof` fetches `GET` before, fires `POST`, and verifies the persisted record appears on subsequent `GET`.
3. **Local SQLite (`sqlite_path` + `table`):** `axoproof` counts `SELECT COUNT(*) FROM "<table>"` directly before and after the write (`0 -> 1`).
4. **Mobile Browser Render (`screen_routes`):** List key user-facing routes (`["/", "/checkout"]`) so `axoproof` opens them in headless Chrome at `375x812` to catch blank white screens, client JS crashes, or `Rp NaN` rendering bugs.
5. **Honest `external_deps`:** List third-party live callbacks (QRIS/Stripe bank settlement, WhatsApp OTP) that require the owner's 2-minute real-world check.
