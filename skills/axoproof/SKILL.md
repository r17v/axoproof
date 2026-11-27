---
name: axoproof
description: "Independent runtime proof, money/privacy safety inspector, structural anti-slop filter, and SHA-256 sealed plain-language Owner Receipt engine (EN/ID) built for non-IT founders, agencies, and AI coding agents. Pairs natively with Axoloom."
license: MIT
---

# AxoProof: Independent Proof & Owner Receipt Protocol

You act under independent inspection for a business owner who judges software strictly by real outcomes: durable saves, exact money, locked private records, and human-grade interfaces.

## 1. Upstream Fact Gate (Kill Slop Before Writing)
1. If `Axoloom` MCP tools (`axoloom_session_boot`, `axoloom_context`) or `.axoloom/index.db` exist, load project facts, brand voice, SQL tables, and guardrails BEFORE writing code or copy.
2. Never invent prices, testimonials, customer counts, phone numbers, emails, or business rules to fill gaps. Ask the owner at most 3 plain questions first.

## 2. Progressive Specialist Modules (Load Only What Applies)
- **Every feature / bugfix:** Read `modules/no-placebo.md` and `modules/live-persistence.md`.
- **Money, cart, pricing, tax, discount, or stock:** Read `modules/cash-register.md`.
- **Login, admin, customer/patient records, API keys, or SQL:** Read `modules/door-locks.md`.
- **Any customer-facing screen, form, or copy:** Read `modules/customer-ready.md` (includes the isolated *Fresh-Eyes Customer Critic* pass).
- **When `axoproof verify` returns exit code `2` (`HOLD_RELEASE`):** Read `modules/self-repair.md`.

## 3. Three-Surface Reporting Discipline (Zero Uninvited Browser Popups)
1. Never write a manual "Verified / Safe" summary in chat without executing the `axoproof` binary.
2. Configure `.axoproof/claims.json` (use `--scope diff` on large legacy repos so inspection focuses on touched files + `Axoloom` callers) and run silently without `--open`:
   `axoproof verify --dir <workspace> --scope diff --sync-axoloom --format markdown`
3. In AI Harness chat, paste the compact 3-row Markdown table from `--format markdown` (with the clickable `file://` link to `.axoproof/RECEIPT.html`) plus the 2-minute human checklist. In pure CLI / Termux, omit `--format markdown` so the 74-column terminal ledger prints directly.
