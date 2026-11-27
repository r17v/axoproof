# Module 3: Door-Locks & Customer Privacy (`door-locks`)

Assume strangers on the public internet will knock on every API route without using your frontend UI:

1. **Lock Every Private Endpoint:** Every route reading or mutating customer, patient, order, invoice, wallet, or admin data must enforce session/token authentication on the server and return HTTP `401` or `403` when unauthenticated.
2. **Enforce Row Ownership (Anti-IDOR):** Never query `WHERE id = $1` alone on user-owned records. Scope queries by the authenticated user or tenant (`WHERE id = $1 AND user_id = $2`).
3. **Cross-Check Against Axoloom Graph:** When `.axoloom/axoloom.db` is present, `axoproof` automatically inspects all indexed endpoints and probes undeclared sensitive routes without credentials. Declare every protected route in `.axoproof/claims.json`.
4. **Parameterized SQL & Server-Only Secrets:** Always bind SQL parameters (`$1`, `?`). Never place service-role keys, database passwords, or private API keys in `NEXT_PUBLIC_*`, `VITE_*`, or client bundles.
