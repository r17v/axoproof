# Module 5: Customer-Ready Screen & Human Copy (`customer-ready`)

Customer-facing pages must look and read like a real human business built them, never a generic AI template.

## 1. Structural Visual Discipline (Anti-Template)
- **Match the Business Kind:** Use warm paper/editorial clarity for clinics, cafes, retail, and documents; reserve dark technical chrome only for developer/media tools that justify it.
- **No Card-in-Card or 3-Clone Grids:** Avoid nesting bordered cards inside bordered containers, avoid 3 identical icon-above-heading boxes when a real product preview or clear table serves better, and never add uppercase eyebrow badges above headings.
- **Mobile Ergonomics (`375px`):** Every interactive button/input needs at least `44x44px` tap area, instant `:focus-visible`, WCAG AA contrast, and `overflow-x: clip`.

## 2. Human Microcopy vs Robotic Prose

| Robotic AI Tell (Rewrite Immediately) | Natural Human Copy (Use Instead) |
| :--- | :--- |
| Vague hype slogans, three-adjective triads, decorative em-dash chains | State the concrete action, time, or price in plain everyday words (`"Orders placed before 3:00 PM ship the same afternoon"`). |
| Leaking `Error 500: PrismaClientKnownRequestError` or `undefined` | `"Your order could not be saved because the connection dropped. Please press Try Again."` |
| Fake `info@example.com`, `+1 (555) 0199`, `Lorem ipsum`, `href="#"`, or made-up `"10,000+ users"` | Use verified business contacts/numbers from `Axoloom` vault, or ask the owner before publishing. |

## 3. Mandatory Fresh-Eyes Customer Critic Pass
Before running final `python3 "<skill-dir>/scripts/axoproof.py" verify` on a customer-facing screen, launch one isolated read-only subagent (or dedicated review pass) prompted strictly as a skeptical first-time buyer reading ONLY the rendered screen text to strip stiff phrasing.
