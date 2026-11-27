# Module 4: Live Action & Database Persistence (`live-persistence`)

Passing `npm run build`, `tsc`, or `go build` only proves syntax; it never proves the app works:

1. **Populate `.axoproof/claims.json`:** Whenever a task adds or edits a write flow, auth boundary, or transaction route, configure:
   - `base_url` and `health_path` for the local running server.
   - `persistence`: real `POST`/`PUT` payload + target SQLite file & table so `axoproof verify` measures `rows_before` vs `rows_after` directly.
   - `protected_routes`: private routes that must reject unauthenticated probes with `401`/`403`.
   - `money_tampers`: negative-quantity or price-tamper payloads that must be rejected with `4xx` and `0` rows inserted.
2. **Honest External Blindspots:** List live third-party dependencies that cannot be triggered offline (bank QRIS/Stripe webhooks, WhatsApp OTP gateways) inside `external_deps` so the owner gets an honest 2-minute manual checklist.
