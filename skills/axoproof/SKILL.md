---
name: axoproof
description: "Independent runtime proof, money/privacy safety inspector, structural anti-slop filter, and SHA-256 sealed plain-language Owner Receipt engine (EN/ID) built for non-IT founders, agencies, and AI coding agents. Pairs natively with Axoloom."
license: MIT
---

# AxoProof: Independent Proof & Owner Receipt Protocol

You act under independent inspection for a business owner who cares about real outcomes, exact money, locked customer data, and natural human interfaces.

## 1. Upstream Fact Gate (Kill Slop at the Root)
1. If `Axoloom` MCP tools (`axoloom_session_boot`, `axoloom_context`) or `.axoloom/index.db` exist, load project facts, brand voice, SQL tables, and guardrails BEFORE writing code or copy.
2. Never invent prices, testimonials, customer counts, phone numbers, emails, or business rules to fill gaps. If missing, ask the owner at most 3 plain questions first.

## 2. Auto-Activated Specialist Modules (Read Only What Applies)
- **Every feature / bugfix:** Read `modules/no-placebo.md` and `modules/live-persistence.md`.
- **Money, cart, pricing, tax, discount, or stock:** Read `modules/cash-register.md`.
- **Login, admin, customer/patient records, API keys, or SQL:** Read `modules/door-locks.md`.
- **Any customer-facing screen, form, or copy:** Read `modules/customer-ready.md` (includes the isolated *Fresh-Eyes Customer Critic* pass).
- **When `axoproof verify` returns exit code `2` (`HOLD_RELEASE`):** Read `modules/self-repair.md`.

## 3. Three-Surface Reporting Discipline (Never Auto-Pop Browser Unasked)
1. You are forbidden from writing a manual "Verified / Safe" checklist in chat without running the `axoproof` binary.
2. Wire `.axoproof/claims.json`, ensure the local server is running when testable, and run silently in the background (NEVER pass `--open` unless the user explicitly asks to open the browser):
   `axoproof verify --dir <workspace> --sync-axoloom --format markdown`
3. In AI Harness chat (Claude, Antigravity, Cursor, Codex), paste the compact 3-row Markdown summary table from `--format markdown` (including the clickable `file://` link to `.axoproof/RECEIPT.html` for optional full dossier viewing or PDF handover) plus the 2-minute human checklist. In pure CLI / Termux, omit `--format markdown` so the 74-column terminal ledger prints directly.
