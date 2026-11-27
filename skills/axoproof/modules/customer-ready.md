# Module 5: Customer-Ready Screen & Human Copy (`customer-ready`)

Ensure customer-facing screens look and read like they were built by a careful human business team, not an AI template generator:

1. **Zero Fake Trust or Sample Placeholders:** Never ship `example.com`, `0812-3456-7890`, `Lorem ipsum`, fictional customer testimonials, or unverified `"Trusted by 10,000+ businesses"` claims. Use real owner facts from `Axoloom` or ask the owner.
2. **Human Microcopy & Safe Error Messages:** Write direct, specific labels and messages using everyday words. Never expose raw server errors (`500`, `PrismaClientKnownRequestError`, `SQLITE_ERROR`, stack traces) to customers, and avoid empty brochure hype phrases.
3. **Physical Mobile Comfort:** Keep interactive tap targets at least `44x44px`, maintain readable WCAG AA text contrast, and prevent horizontal page overflow on mobile widths (`320px`).
4. **Isolated Fresh-Eyes Customer Critic Pass:** After technical checks pass on a user-facing page, run a separate read-only subagent (or dedicated review pass) acting strictly as a first-time skeptical customer reading ONLY the rendered text/UI to rewrite any stiff robotic sentences before generating the final receipt.
