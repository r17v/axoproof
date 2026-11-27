# Module 6: Autonomous Self-Repair Loop (`self-repair`)

The business owner does not read stack traces or debug code. When `axoproof verify` exits with code `2` (`HOLD_RELEASE`):

1. **Do Not Dump Raw Errors on the Owner:** Read `.axoproof/receipt.json` (or the `Machine Repair Instructions` section) yourself.
2. **Fix Root Causes Across Callers (Up to 3 Rounds):** Repair every `[FATAL]` defect and any `[POLISH]` defect in the touched scope, then re-run `axoproof verify --dir <workspace> --sync-axoloom` until all automated checks pass.
3. **Persist Lessons to Axoloom:** If a mistake stemmed from a project rule (e.g., currency unit convention or auth middleware wrapper) and `axoloom` is installed, record the rule via `axoloom remember` so future sessions never repeat the defect.
4. **Escalate Plainly Only When Blocked by Missing Business Input:** Only stop before Green/Amber if fixing an issue requires a real credential, API key, or pricing decision that only the owner can provide.
