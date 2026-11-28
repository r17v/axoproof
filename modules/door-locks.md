# Module 3: Door-Locks & Customer Privacy (`door-locks`)

Assume strangers on the internet will call every API route directly without touching your frontend UI.

## Wrong vs Right Patterns

| Open-Door Vulnerability (Forbidden) | Locked Boundary Pattern (Required) |
| :--- | :--- |
| Checking login only in React/Next client router while `/api/customers` has no server check | Verify session/JWT inside every API route, Server Action, or middleware FIRST; return HTTP `401` when unauthenticated. |
| `SELECT * FROM orders WHERE id = $1` (IDOR leak across customers) | Scope every read/update/delete by owner or tenant: `WHERE id = $1 AND user_id = $2` (or enforce verified Supabase RLS `auth.uid() = user_id` on every table). |
| String-interpolated SQL (`SELECT ... WHERE email = '${email}'`) | Parameterized queries only (`$1`, `?`, or typed ORM bindings). |
| `NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY` or `VITE_SECRET_KEY` | Keep service-role keys, payment secrets, and DB URLs strictly in server-only env vars; never serialize full user rows (password hashes, tokens) in JSON responses. |

## Axoloom Anti-Cheat Cross-Check
When `.axoloom/index.db` exists, `axoproof` automatically discovers all indexed `/api/*` routes and knocks on sensitive endpoints without login. Declare every private endpoint in `.axoproof/claims.json`.
