# Module 2: Cash-Register & Exact Money (`cash-register`)

Treat every price, invoice, wallet, discount, tax split, and inventory count as real business cash.

## Wrong vs Right Patterns

| Cash-Register Bug (Forbidden) | Exact Ledger Pattern (Required) |
| :--- | :--- |
| `price FLOAT` / `float64` / `parseFloat(req.body.price)` | Store & compute integer minor units (`BIGINT` / `int64` Rupiah or cents) or SQL `NUMERIC(18,2)`. Compute splits/tax via integer math with explicit remainder allocation. |
| `total = req.body.quantity * req.body.price` | Ignore client-submitted prices/discounts. Query authoritative `unit_price` and active promo rules from the server database by `sku` / `product_id`. |
| Accepting `quantity: 0` or `quantity: -3` | Enforce `quantity >= 1` and `stock >= quantity` (`UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1`) inside the transaction, returning HTTP `400`/`422` otherwise. |
| Double charge when customer taps "Pay" twice | Require an `idempotency_key` (`UNIQUE` index on `(user_id, idempotency_key)`) and wrap order + ledger rows in one atomic `BEGIN ... COMMIT` transaction. |
