# Module 1: Anti-Placebo & Real Wiring (`no-placebo`)

Never hand the owner a painted facade that looks finished on screen while doing nothing underneath:

1. **Zero Dead Controls:** Every visible `<button>`, `<form>`, filter, toggle, and menu link must execute a real handler. Empty handlers (`onClick={() => {}}`), `href="#"` placeholders, and `setTimeout` timers pretending to save data are strictly forbidden.
2. **Zero Hardcoded Business Metrics:** Dashboards, summary cards, and report routes must compute figures from real database queries. Never hardcode fake revenue (`Rp 18.500.000`, `$12,400`), sample order arrays (`mockOrders`), or fake chart arrays in production routes/components.
3. **Zero Silenced Failures:** Never add `.skip` / `t.Skip` to hide a failing test, and never swallow database/API write errors while showing a green success toast to the user.
