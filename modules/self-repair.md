# Module 6: Autonomous Self-Repair Loop (`self-repair`)

Non-technical business owners do not debug stack traces. When `axoproof.py verify` exits with code `2` (`HOLD_RELEASE`):

1. **Absorb Technical Errors Internally:** Read `.axoproof/receipt.json` (or the `Technical repair slip` block) without dumping raw compiler/SQL logs on the owner.
2. **Trace Callers Before Editing:** If `Axoloom` (`axoloom_blast_radius`) is available, inspect upstream callers of the broken route/table so fixing one defect does not break an adjacent flow.
3. **Iterate Up to 3 Rounds Until Proven:** Fix every `[BLOCKING]` defect and touched `[POLISH]` item, then re-run `python3 "<skill-dir>/scripts/axoproof.py" verify --dir <workspace> --scope diff --sync-axoloom` until verdict reaches `SAFE_TO_TEST` (or `LOCAL_ONLY_MANUAL_CHECK` when no local server exists).
4. **Compound Memory into Axoloom:** Record recurring project constraints (e.g., currency minor-unit rule or auth wrapper name) via `axoloom remember` so future sessions never repeat the mistake.
5. **Escalate Only Business Decisions:** Stop and ask the owner in plain words only when blocked by a missing live API credential or business policy choice.
