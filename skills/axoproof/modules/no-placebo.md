# Module 1: Anti-Placebo & Real Wiring (`no-placebo`)

Never hand the owner a painted facade that looks finished on screen while doing nothing underneath.

## Wrong vs Right Patterns

| Placebo Trap (Forbidden) | Production Wiring (Required) |
| :--- | :--- |
| `<button onClick={() => toast("Saved!")}>` or `onClick={() => {}}` | Disable button while pending, send `POST`/`PATCH` to real route, show toast ONLY after `2xx` response, and refresh persisted state. |
| `setTimeout(() => setSuccess(true), 800)` | Await real database/API write; surface clear human error state if network or validation fails. |
| Hardcoded dashboard KPIs (`Rp 18.500.000`, `const mockOrders = [...]`) | Query `SUM(...)` / `COUNT(...)` from real database tables; render `Rp 0` or an honest empty state when zero rows exist. |
| Silencing broken tests with `it.skip(...)` or `t.Skip()` | Fix the underlying handler/schema defect or update the outdated fixture so the test runs and passes cleanly. |

## Pre-Delivery Check
1. Trace every new button, form, filter, and modal action from UI event -> API route -> database write -> UI state reload.
2. Verify that reloading the browser page (`F5`) keeps the saved data on screen rather than losing in-memory React/JS state.
