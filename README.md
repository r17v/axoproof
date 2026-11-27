<p align="center">
  <img src="assets/brand/axoproof-horizontal.svg" alt="AxoProof" width="330">
</p>

<h3 align="center">
  Independent Runtime Proof, Money/Privacy Inspector, Structural Anti-Slop Filter,<br>
  and SHA-256 Sealed Owner Receipt (<code>EN / ID</code>) for AI Coding Agents.
</h3>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-5C45B9.svg" alt="MIT License"></a>
  <img src="https://img.shields.io/badge/Go-1.26%2B_Pure_Zero--CGo-00ADD8.svg" alt="Pure Go Zero CGo">
  <img src="https://img.shields.io/badge/Skill-1_Door_%2B_6_Auto_Modules-10b981.svg" alt="1 Door Skill + 6 Modules">
  <a href="https://github.com/r17v/axoloom"><img src="https://img.shields.io/badge/Axoloom-2--Way_Graph_Bridge-d97706.svg" alt="Axoloom Compatible"></a>
</p>

<p align="center">
  <img src="assets/showcase/receipt-showcase.png" alt="AxoProof Sealed Bilingual Owner Receipt catching AI placebo behavior and verifying real SQLite row persistence" width="100%">
</p>

## Why AxoProof Exists

AI coding agents routinely tell non-technical founders, agency owners, and busy builders *"Done! Build passed!"* while leaving behind hidden landmines:

- **Placebo Interfaces:** Buttons that show a green success toast while saving `0` rows to the database, or dashboard cards showing hardcoded sample revenue (`Rp 18.500.000`).
- **Cash-Register Bugs:** Floating-point rounding errors on currency, negative-quantity exploits (`quantity: -5`), or checkout routes that trust item prices sent directly from the buyer's browser.
- **Unlocked Customer Doors:** Sensitive `/api/customers` or `/api/orders` endpoints left open without authentication, or secret keys leaked into `NEXT_PUBLIC_*` / `VITE_*` client bundles.
- **Customer-Screen AI Slop:** Leftover `example.com` contacts, dead `href="#"` links, raw `500` stack traces shown to customers, and stiff brochure cliches.

An AI coding agent grading its own work has a conflict of interest. **AxoProof** separates the **Inspector** from the **Builder**: one thin Agent Skill (`<= 45` lines) activates **6 specialist modules** during coding, and a standalone zero-CGo binary runs counterfactual live HTTP + database probes and stamps a tamper-evident **`.axoproof/RECEIPT.html`** (with instant **English / Bahasa Indonesia** toggle) that the AI cannot forge.

---

## Synergy with [Axoloom](https://github.com/r17v/axoloom)

`Axoloom` and `AxoProof` form a closed quality loop across your workspace:

1. **Upstream (`Axoloom` -> `AxoProof`):** Before writing code or copy, the agent loads real business facts, brand voice, and permanent guardrails from `Axoloom` so it never invents generic filler. During verification, `AxoProof` reads `.axoloom/axoloom.db` directly to catch sensitive API routes even if the agent forgot to declare them in `claims.json`.
2. **Downstream (`AxoProof` -> `Axoloom`):** Running `axoproof verify --sync-axoloom` writes the SHA-256 sealed verification outcome back into `Axoloom` handoff memory so future sessions inherit lessons automatically.

---

## 1 Skill Entrypoint, 6 Specialist Inspectors Inside

Non-technical owners never need to memorize 10 slash commands. Installing the single `axoproof` skill automatically orchestrates 6 focused modules:

| Module | What It Guards | Receipt Track |
| :--- | :--- | :--- |
| `no-placebo` | Stops empty `onClick` handlers, simulated `setTimeout` saves, hardcoded fake KPI figures, and `.skip` tests | **Fatal Business Risk** |
| `cash-register` | Enforces integer minor-unit currency (`int64` / `NUMERIC`), server-side price lookup, and negative-quantity rejection | **Fatal Business Risk** |
| `door-locks` | Verifies authentication on private routes, blocks raw SQL string concatenation, and catches exposed client secrets | **Fatal Business Risk** |
| `live-persistence` | Counts real database rows before vs after live HTTP requests (`0 -> 1`) instead of trusting `build` logs | **Live Runtime Proof** |
| `customer-ready` | Catches `example.com` / fake testimonials, dead `#` links, leaked stack traces, brochure slop, `<44px` mobile buttons, plus runs an isolated *Fresh-Eyes Customer Critic* pass | **Customer Polish Track** |
| `self-repair` | Forces the AI agent to fix `[FATAL]` defects up to 3 rounds autonomously before handing the receipt to the owner | **Autonomous Repair** |

---

## Quick Start

### 1. Install Binary + Agent Skill

#### macOS (Apple Silicon & Intel) / Linux / Android Termux

```bash
curl -fsSL https://raw.githubusercontent.com/r17v/axoproof/main/install.sh | sh
```

#### Windows 10 / 11 (`x64` & `ARM64` PowerShell)

```powershell
irm https://raw.githubusercontent.com/r17v/axoproof/main/install.ps1 | iex
```

#### Build from Source (`Go 1.26+`, Zero CGo)

```bash
git clone https://github.com/r17v/axoproof.git
cd axoproof
CGO_ENABLED=0 go build -o ~/.local/bin/axoproof ./cmd/axoproof
mkdir -p ~/.agents/skills && cp -R ./skills/axoproof ~/.agents/skills/axoproof
```

### 2. Try the 1-Command Sabotage vs Verified Demo

AxoProof automatically starts each demo server via `start_command`, probes live HTTP/SQLite persistence + headless mobile `375px` screen rendering, and shuts the server down cleanly:

**Catch 7 blocking AI vibe-coding lies red-handed (`HOLD_RELEASE`):**

```bash
./axoproof verify --dir ./examples/sabotaged-checkout --lang id || true
```

**Verify the repaired store passing all live DB, auth, tamper, and mobile screen probes (`SAFE_TO_TEST`):**

```bash
./axoproof verify --dir ./examples/verified-checkout --lang id
```

---

## Three Surfaces, Zero Forced Popups

AxoProof adapts cleanly to where you work without hijacking your screen with uninvited browser tabs:

1. **Terminal / SSH / Android Termux (`--format table`, default):** Prints a 74-column structured inspection ledger directly in the terminal showing every plain-language finding, live `HTTP` / database row measurement (`0 -> 1`), and SHA-256 seal.
2. **AI Coding Harnesses (`--format markdown` inside Claude, Antigravity, Cursor, Codex):** Runs silently in the background during self-repair loops, then outputs a compact 3-row Markdown summary table in chat with a clickable `file://` link to `.axoproof/RECEIPT.html`.
3. **Archival Browser Dossier & PDF Handover (`--open` or click the link):** Opens the bilingual (`EN / ID`), `Paper / Night` Swiss inspection sheet in your browser when you want a calm full-page review or a clean PDF certificate for client project handover.

---

## CLI Reference

```bash
axoproof init       [--dir <path>]
axoproof verify     [--dir <path>] [--claims <path>] [--scope all|diff] [--lang en|id] [--format table|markdown|json] [--sync-axoloom] [--open]
axoproof check-seal [--dir <path>] [<receipt.json>]
axoproof version
```

- Exit code `0`: `SAFE_TO_TEST` or `LOCAL_ONLY_MANUAL_CHECK`
- Exit code `2`: `HOLD_RELEASE` (blocks premature agent completion or CI merge; tamper check failed)

---

## License

Licensed under the [MIT License](./LICENSE). Third-party attributions are documented in [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).
