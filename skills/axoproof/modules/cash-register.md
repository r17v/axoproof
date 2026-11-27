# Module 2: Cash-Register & Exact Money (`cash-register`)

Treat every price, invoice, wallet, discount, tax split, and inventory count as real business cash:

1. **Integer Minor Units Only:** Store and calculate currency in integer minor units (`int64` Rupiah / cents) or exact database `NUMERIC` / `DECIMAL`. Never use `float32`, `float64`, `REAL`, `DOUBLE PRECISION`, or `parseFloat` on money.
2. **Server-Authoritative Pricing:** Checkout and order endpoints must look up unit prices, discounts, and tax rates from the server database by item ID/SKU. Never trust `price`, `total`, or `discount` sent from the client browser.
3. **Reject Negative & Zero Exploits:** Validate `quantity > 0` and non-negative totals on the server before touching inventory or ledgers, returning HTTP `400` or `422` on invalid payloads.
4. **Atomic Writes & Idempotency:** Wrap multi-table order + payment/ledger writes inside a single database transaction and guard payment capture endpoints against duplicate double-click submissions.
