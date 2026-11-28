#!/usr/bin/env python3
"""AxoProof: Independent runtime proof, money/privacy inspector, structural anti-slop filter,
and SHA-256 sealed Owner Inspection Dossier for AI coding agents.

100% Python 3.10+ standard library only. Zero external packages, zero network telemetry.
Pairs natively with Axoloom (.axoloom/index.db).
"""

from __future__ import annotations

import argparse
import hashlib
import html
import ipaddress
import json
import os
import platform
import re
import shutil
import signal
import socket
import sqlite3
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
import webbrowser
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

VERSION = "v0.2.0"

VERDICT_SAFE = "SAFE_TO_TEST"
VERDICT_MANUAL = "LOCAL_ONLY_MANUAL_CHECK"
VERDICT_HOLD = "HOLD_RELEASE"

TRACK_FATAL = "FATAL_BUSINESS_RISK"
TRACK_POLISH = "CUSTOMER_POLISH"

VALID_SQL_IDENT = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")

# Static Scanner Patterns
RE_EMPTY_CLICK = re.compile(r"on(Click|Submit)\s*=\s*\{\s*\([^)]*\)\s*=>\s*\{\s*(e\.preventDefault\(\)\s*;?\s*)?\}\s*\}")
RE_FAKE_DELAY = re.compile(r"setTimeout\s*\(\s*\([^)]*\)\s*=>\s*\{[^}]*(saved|success|berhasil|paid|terkirim)", re.IGNORECASE)
RE_HARDCODED_METRIC = re.compile(
    r"(total\s+(revenue|sales|income|omzet|pendapatan)|today['\u2019]?s\s+(revenue|sales)|pendapatan\s+hari\s+ini)"
    r"[^<\n]{0,80}(Rp\s?[0-9][0-9.,]{4,}|\$[0-9][0-9,.]{3,})",
    re.IGNORECASE,
)
RE_MOCK_PROD = re.compile(r"\b(mockOrders|mockUsers|mockPayments|fakeTransactions|DUMMY_ORDERS|FAKE_REVENUE)\b")
RE_SKIPPED_TEST = re.compile(r"\b(it|test|describe)\.skip\s*\(|^\s*t\.Skip\s*\(|@unittest\.skip")

RE_SQL_FLOAT_MONEY = re.compile(
    r"\b(price|unit_price|amount|total|subtotal|balance|tax|discount|fee|harga|tagihan|saldo)\b"
    r"[^,\n;]{0,40}\b(FLOAT|REAL|DOUBLE\s+PRECISION|float64|float32)\b",
    re.IGNORECASE,
)
RE_GO_FLOAT_MONEY = re.compile(
    r"\b(price|unitPrice|amount|total|subtotal|balance|tax|discount|fee|harga|saldo)[A-Za-z0-9_]*\s+float(32|64)\b",
    re.IGNORECASE,
)
RE_JS_FLOAT_MONEY = re.compile(r"parseFloat\s*\([^)]*(price|amount|total|subtotal|discount|tax|harga|saldo)", re.IGNORECASE)
RE_PY_FLOAT_MONEY = re.compile(r"\b(price|amount|total|subtotal|discount|tax|harga|saldo)\s*=\s*float\s*\(", re.IGNORECASE)
RE_CLIENT_PRICE = re.compile(
    r"(req\.body|request\.json|request\.data|body|payload|input)\s*(\.|\[['\"])\s*(price|unit_price|total_amount|final_price|harga)\b",
    re.IGNORECASE,
)

RE_PUBLIC_SECRET = re.compile(r"(NEXT_PUBLIC_|VITE_|REACT_APP_|PUBLIC_)[A-Z0-9_]*(SECRET|SERVICE_ROLE|PRIVATE_KEY|DB_PASSWORD|ADMIN_KEY)")
RE_LIVE_KEY = re.compile(r"\b(sk_live_[A-Za-z0-9]{16,}|rk_live_[A-Za-z0-9]{16,}|ghp_[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16})\b")
RE_RAW_SQL = re.compile(
    r'(fmt\.Sprintf\s*\(\s*"\s*(SELECT|INSERT|UPDATE|DELETE)\b[^"]*%[sv]'
    r'|`\s*(SELECT|INSERT|UPDATE|DELETE)\b[^`]*\$\{'
    r'|f["\']\s*(SELECT|INSERT|UPDATE|DELETE)\b[^"\']*\{)',
    re.IGNORECASE,
)
RE_ROUTE_DECL = re.compile(r"""(HandleFunc|get|post|put|patch|delete|route)\s*\(\s*["'`](/api/[a-zA-Z0-9_/\-:]+)["'`]""", re.IGNORECASE)
RE_SENSITIVE_ROUTE = re.compile(r"^/api/(admin|customers|users|orders|invoices|patients|payments|wallets|transactions|ledger)\b", re.IGNORECASE)
RE_AUTH_EVIDENCE = re.compile(
    r"\b(auth|session|jwt|bearer|token|cookie|current_user|currentUser|user_id|requireAuth|withAuth|middleware|Unauthorized|401|403)\b",
    re.IGNORECASE,
)

RE_FAKE_CONTACT = re.compile(
    r"(info@example\.com|support@example\.com|test@example\.com|lorem\s+ipsum|0812-?3456-?7890|\+1\s*\(?555\)?[\s-]*01[0-9]{2})",
    re.IGNORECASE,
)
RE_FAKE_SOCIAL = re.compile(
    r"(trusted\s+by\s+(10,000|50,000|100,000)\+|dipercaya\s+oleh\s+(10\.000|50\.000|100\.000)\+)",
    re.IGNORECASE,
)
RE_DEAD_ANCHOR = re.compile(r"""href\s*=\s*["']#["']""")
RE_RAW_STACK = re.compile(r"\b(err\.stack|error\.stack|traceback\.format_exc\(\)|debug\.Stack\(\))\b")
RE_ROBOTIC_COPY = re.compile(
    r"\b(revolutionize\s+your|unleash\s+the\s+power|seamlessly\s+integrated|in\s+today['\u2019]?s\s+fast-paced\s+digital)\b",
    re.IGNORECASE,
)
RE_TINY_BUTTON = re.compile(
    r'(<button[^>]*class(Name)?=["\'][^"\']*\b(h-5|h-6|w-5|w-6)\b|<button[^>]*style=["\'][^"\']*height:\s*(1[6-9]|2[0-9]|3[0-2])px)'
)

RE_CRASH_OVERLAY = re.compile(
    r"(Application error:\s*a client-side exception|Unhandled Runtime Error|Cannot read properties of (undefined|null)|Hydration failed because the initial UI|Internal Server Error)",
    re.IGNORECASE,
)
RE_NAN_CURRENCY = re.compile(r"(\bRp\s*(NaN|undefined|null)\b|\$(NaN|undefined|null)\b|\bNaN%\b)")
RE_STRIP_HTML = re.compile(r"<script\b[^>]*>.*?</script>|<style\b[^>]*>.*?</style>|<[^>]+>", re.DOTALL | re.IGNORECASE)

IGNORED_DIRS = {
    ".git", ".axoproof", ".axoloom", "node_modules", "dist", "build",
    ".next", "vendor", "__pycache__", ".venv", "venv", ".pytest_cache",
}
SUPPORTED_EXTS = {".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".sql", ".prisma", ".html", ".env"}

AXOPROOF_ICON_PATH = (
    "M 156.260 519.877 C 144.424 517.940 133.342 509.503 127.774 498.193 C 124.761 492.073 124.502 490.782 124.524 482.021 "
    "C 124.552 471.261 125.763 467.694 133.080 456.830 C 145.628 438.198 168.398 423.004 193.830 416.292 C 202.073 414.117 "
    "206.327 413.669 222.655 413.254 C 243.482 412.724 251.927 413.430 279.339 417.989 C 310.529 423.176 322.202 424.314 "
    "338.000 423.707 C 355.370 423.040 365.136 421.256 380.604 415.922 C 394.905 410.991 403.843 406.556 415.473 398.622 "
    "C 443.514 379.490 464.035 348.511 471.680 313.768 C 473.278 306.510 473.500 300.432 473.500 264.000 C 473.500 224.673 "
    "473.380 221.907 471.204 211.166 C 467.037 190.600 460.490 173.216 450.839 157.090 C 434.725 130.165 412.329 108.442 "
    "386.872 95.045 C 364.661 83.356 346.728 78.656 324.500 78.697 C 291.076 78.759 262.959 90.557 239.831 114.223 "
    "C 228.700 125.613 226.555 126.620 216.808 125.039 C 207.606 123.546 191.477 123.800 184.000 125.556 C 180.425 126.396 "
    "173.675 128.921 169.000 131.168 C 162.042 134.512 158.955 136.817 151.982 143.876 C 142.729 153.244 137.859 161.256 "
    "133.979 173.500 C 132.206 179.095 131.628 184.011 131.098 198.000 C 130.249 220.429 129.014 223.013 116.490 228.543 "
    "C 96.668 237.297 80.366 255.316 74.634 274.809 C 69.508 292.241 72.934 313.352 83.518 329.556 C 88.815 337.665 "
    "100.232 348.725 107.927 353.202 C 111.427 355.237 117.713 358.094 121.895 359.549 C 138.034 365.163 141.650 367.433 "
    "145.044 374.086 C 147.233 378.377 147.699 388.006 145.944 392.685 C 142.620 401.548 132.891 409.761 121.684 413.165 "
    "C 113.673 415.598 104.786 415.235 91.823 411.944 C 41.061 399.057 3.281 351.580 0.364 297.007 C -1.508 262.010 "
    "16.503 221.767 45.000 197.275 C 51.536 191.657 61.119 185.184 70.661 179.940 C 73.212 178.538 73.631 177.544 "
    "74.795 170.148 C 81.220 129.292 106.792 93.289 142.000 75.527 C 157.519 67.698 176.341 63.000 192.191 63.000 "
    "L 201.161 63.000 211.235 52.449 C 240.047 22.271 275.980 5.145 319.335 0.927 C 340.359 -1.118 368.732 2.412 "
    "392.500 10.029 C 424.249 20.204 451.376 37.300 478.065 63.954 C 523.518 109.348 549.359 163.087 555.025 224.000 "
    "C 555.736 231.644 555.995 276.237 555.797 357.000 L 555.500 478.500 552.276 485.089 C 537.582 515.116 492.521 515.140 "
    "477.741 485.128 C 474.507 478.561 474.499 478.507 474.000 459.561 L 473.500 440.575 466.566 446.863 C 445.853 465.646 "
    "412.666 481.985 382.134 488.431 C 353.423 494.493 328.367 494.053 288.000 486.775 C 268.705 483.297 263.363 482.703 "
    "250.500 482.609 C 232.632 482.479 223.624 484.194 211.346 490.065 C 204.248 493.459 202.046 495.210 194.346 503.585 "
    "C 189.481 508.877 183.700 514.248 181.500 515.521 C 174.558 519.539 164.645 521.250 156.260 519.877 Z "
    "M 281.000 400.694 C 237.750 384.211 199.193 347.153 182.985 306.488 C 172.388 279.902 170.032 263.528 170.039 216.500 "
    "C 170.045 180.183 170.181 179.424 177.923 172.798 C 182.424 168.945 188.243 167.000 195.267 167.000 C 202.599 167.000 "
    "217.892 163.999 229.500 160.282 C 245.163 155.267 260.751 147.345 273.182 138.082 C 287.424 127.468 289.162 126.633 "
    "297.000 126.633 C 304.543 126.633 306.970 127.689 318.500 135.984 C 342.596 153.321 370.695 164.138 398.950 166.954 "
    "C 405.247 167.582 411.874 168.795 413.676 169.651 C 418.240 171.816 423.765 178.231 424.517 182.237 C 424.863 184.083 "
    "424.961 203.347 424.734 225.047 C 424.285 267.947 424.031 270.257 417.470 290.941 C 403.323 335.540 371.494 372.676 "
    "329.241 393.879 C 307.075 405.003 296.299 406.525 281.000 400.694 Z M 304.240 366.634 C 321.647 360.082 334.954 351.248 "
    "349.500 336.587 C 368.136 317.804 378.400 299.614 385.673 272.482 C 388.240 262.908 388.366 261.249 388.760 231.870 "
    "L 389.171 201.239 379.919 199.539 C 354.226 194.817 329.775 184.623 306.295 168.842 C 302.332 166.179 298.395 164.000 "
    "297.545 164.000 C 296.695 164.000 291.837 166.876 286.750 170.391 C 263.722 186.302 241.592 195.157 211.000 200.701 "
    "L 206.500 201.516 206.562 228.508 C 206.632 258.906 207.661 267.253 213.482 284.651 C 220.559 305.802 230.189 321.379 "
    "245.905 337.095 C 259.867 351.057 275.380 361.271 291.500 367.115 C 297.768 369.388 296.830 369.423 304.240 366.634 Z "
    "M 277.500 314.087 C 273.275 312.112 244.728 286.579 241.757 282.118 C 239.570 278.835 239.429 270.005 241.499 266.002 "
    "C 243.533 262.068 249.056 258.027 253.351 257.330 C 260.610 256.152 263.547 257.570 273.505 267.056 C 278.669 271.975 "
    "283.468 276.000 284.171 276.000 C 284.874 276.000 296.711 264.788 310.475 251.085 C 331.978 229.676 336.098 226.011 "
    "339.750 225.046 C 344.798 223.712 350.308 224.881 354.495 228.175 C 357.693 230.691 360.946 237.267 360.978 241.282 "
    "C 361.008 245.034 358.627 250.484 355.756 253.240 C 354.461 254.483 340.073 268.631 323.783 284.680 C 299.952 308.159 "
    "293.416 314.068 290.332 314.925 C 284.943 316.421 282.097 316.235 277.500 314.087 Z"
)


# ---------------------------------------------------------------------------
# Claims Manifest
# ---------------------------------------------------------------------------

def default_claims_path(work_dir: Path) -> Path:
    return work_dir / ".axoproof" / "claims.json"


def load_claims(work_dir: Path, explicit_path: str = "") -> dict[str, Any]:
    target = Path(explicit_path) if explicit_path.strip() else default_claims_path(work_dir)
    if not target.exists():
        return {
            "project_name": work_dir.resolve().name,
            "summary": "Zero-config workspace inspection",
            "persistence": [],
            "protected_routes": [],
            "money_tampers": [],
            "screen_routes": [],
            "external_deps": [],
        }
    data = json.loads(target.read_text(encoding="utf-8"))
    if not str(data.get("project_name", "")).strip():
        data["project_name"] = work_dir.resolve().name
    data.setdefault("summary", "")
    data.setdefault("persistence", [])
    data.setdefault("protected_routes", [])
    data.setdefault("money_tampers", [])
    data.setdefault("screen_routes", [])
    data.setdefault("external_deps", [])
    return data


def write_claims_template(work_dir: Path) -> Path:
    p = default_claims_path(work_dir)
    p.parent.mkdir(parents=True, exist_ok=True)
    sample = {
        "project_name": work_dir.resolve().name,
        "summary": "Describe in one plain sentence what feature was built for the business owner.",
        "base_url": "http://127.0.0.1:3000",
        "start_command": "npm run start",
        "health_path": "/api/health",
        "touches_money": True,
        "touches_auth": True,
        "screen_routes": ["/"],
        "persistence": [
            {
                "name": "Create order persists across read-back (Postgres/Supabase/SQLite)",
                "method": "POST",
                "path": "/api/orders",
                "payload_json": '{"sku":"ITEM-AXO-01","quantity":1}',
                "read_back_path": "/api/orders",
                "expect_read_substring": "ITEM-AXO-01",
                "expect_min_delta": 1,
            }
        ],
        "protected_routes": [
            {
                "name": "Customer order list blocks unauthenticated strangers",
                "method": "GET",
                "path": "/api/customers/orders",
            }
        ],
        "money_tampers": [
            {
                "name": "Checkout rejects negative quantity exploit",
                "method": "POST",
                "path": "/api/orders",
                "tampered_body_json": '{"sku":"ITEM-AXO-01","quantity":-3,"price":0.01}',
            }
        ],
        "external_deps": [
            {
                "service": "Payment Gateway Live Settlement",
                "note_en": "Complete one real small payment on staging/production to confirm bank settlement callbacks.",
                "note_id": "Lakukan 1 transaksi kecil nyata di staging/produksi untuk memastikan callback pembayaran masuk.",
            }
        ],
    }
    p.write_text(json.dumps(sample, indent=2) + "\n", encoding="utf-8")
    return p


# ---------------------------------------------------------------------------
# Static Workspace Scanner + Git Diff Scope
# ---------------------------------------------------------------------------

def is_test_file(rel: str) -> bool:
    low = rel.lower()
    base = os.path.basename(low)
    return (
        low.endswith("_test.go")
        or ".test." in low
        or ".spec." in low
        or base.startswith("test_")
        or "/__tests__/" in low
        or low.startswith("tests/")
    )


def resolve_git_diff_files(work_dir: Path) -> set[str]:
    changed: set[str] = set()
    try:
        proc = subprocess.run(
            ["git", "-C", str(work_dir), "status", "--porcelain"],
            capture_output=True, text=True, timeout=5, check=False,
        )
        if proc.returncode == 0:
            for line in proc.stdout.splitlines():
                if len(line) < 4:
                    continue
                raw = line[3:].strip()
                if " -> " in raw:
                    raw = raw.split(" -> ", 1)[1].strip()
                raw = raw.strip('"')
                if raw:
                    changed.add(raw.replace("\\", "/"))
    except Exception:
        pass

    if not changed:
        try:
            proc = subprocess.run(
                ["git", "-C", str(work_dir), "diff", "--name-only", "HEAD~1", "HEAD"],
                capture_output=True, text=True, timeout=5, check=False,
            )
            if proc.returncode == 0:
                for line in proc.stdout.splitlines():
                    p = line.strip()
                    if p:
                        changed.add(p.replace("\\", "/"))
        except Exception:
            pass
    return changed


def scan_workspace(work_dir: Path, allowed_rel: set[str] | None = None) -> dict[str, Any]:
    files_scanned = 0
    fatal_risks: list[dict[str, Any]] = []
    polish_warnings: list[dict[str, Any]] = []
    discovered_urls: list[str] = []
    seen_routes: set[str] = set()

    root = work_dir.resolve()
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in IGNORED_DIRS]
        for fname in sorted(filenames):
            full = Path(dirpath) / fname
            ext = full.suffix.lower()
            if ext not in SUPPORTED_EXTS and not fname.startswith(".env"):
                continue
            try:
                if full.stat().st_size > 2 * 1024 * 1024:
                    continue
            except OSError:
                continue

            rel = full.relative_to(root).as_posix()
            if rel in {"scripts/axoproof.py", "tests/test_axoproof.py"}:
                continue
            if allowed_rel is not None and rel not in allowed_rel:
                continue

            try:
                text = full.read_text(encoding="utf-8", errors="ignore")
            except OSError:
                continue

            files_scanned += 1
            test_file = is_test_file(rel)
            has_file_auth = bool(RE_AUTH_EVIDENCE.search(text))

            for line_no, raw_line in enumerate(text.splitlines(), start=1):
                trimmed = raw_line.strip()
                if not trimmed or trimmed.startswith(("//", "#", "--")):
                    continue

                m_route = RE_ROUTE_DECL.search(raw_line)
                if m_route:
                    route_path = m_route.group(2)
                    if route_path not in seen_routes:
                        seen_routes.add(route_path)
                        discovered_urls.append(route_path)
                    if not test_file and RE_SENSITIVE_ROUTE.search(route_path) and not has_file_auth:
                        fatal_risks.append({
                            "id": "DOOR_OPEN_SENSITIVE_ROUTE",
                            "track": TRACK_FATAL,
                            "module": "door_locks",
                            "file": rel,
                            "line": line_no,
                            "title_en": "Customer or admin data endpoint has no login lock",
                            "title_id": "Pintu data pelanggan atau admin terbuka tanpa kunci login",
                            "business_impact_en": f"Anyone with the link can read or modify sensitive records ({route_path}) without logging in.",
                            "business_impact_id": f"Orang luar yang tahu alamat link ({route_path}) bisa mengintip atau mengubah data pelanggan tanpa perlu login.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Enforce session/JWT authentication middleware and verify resource ownership (user_id/tenant_id) on {route_path} in {rel}.",
                        })

                if test_file:
                    if RE_SKIPPED_TEST.search(raw_line):
                        fatal_risks.append({
                            "id": "PLACEBO_SKIPPED_TEST",
                            "track": TRACK_FATAL,
                            "module": "no_placebo",
                            "file": rel,
                            "line": line_no,
                            "title_en": "Automated test was silenced instead of fixed",
                            "title_id": "Pengujian otomatis sengaja dimatikan (skip) oleh AI",
                            "business_impact_en": "A failing check was hidden with skip, meaning broken behavior can slip through undetected.",
                            "business_impact_id": "Pengujian yang gagal disembunyikan dengan fitur skip sehingga bug asli tidak ketahuan.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Remove skip decorator/call in {rel} and make the underlying test pass against real logic.",
                        })
                    continue

                # 1. Anti-Placebo
                if RE_EMPTY_CLICK.search(raw_line):
                    fatal_risks.append({
                        "id": "PLACEBO_DEAD_BUTTON",
                        "track": TRACK_FATAL,
                        "module": "no_placebo",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Display-only button or form does nothing when clicked",
                        "title_id": "Tombol atau formulir pajangan (diklik tidak melakukan apa-apa)",
                        "business_impact_en": "Customers can click this button on screen, but no action or data save is wired behind it.",
                        "business_impact_id": "Pelanggan bisa memencet tombol ini di layar, tetapi tidak ada proses penyimpanan apa pun di belakangnya.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Wire the onClick/onSubmit handler in {rel} to a real API mutation or state action and verify persistence.",
                    })
                if RE_FAKE_DELAY.search(raw_line):
                    fatal_risks.append({
                        "id": "PLACEBO_FAKE_SAVE_TIMER",
                        "track": TRACK_FATAL,
                        "module": "no_placebo",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Simulated loading timer pretending to save data",
                        "title_id": "Efek loading pura-pura menyimpan data (bohongan)",
                        "business_impact_en": "The interface waits briefly and shows a success message without saving customer data.",
                        "business_impact_id": "Layar hanya menunggu sebentar lalu menampilkan pesan berhasil, padahal data tidak disimpan.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Replace the fake setTimeout success toast in {rel} with a real backend API call and database write.",
                    })
                if RE_HARDCODED_METRIC.search(raw_line) or RE_MOCK_PROD.search(raw_line):
                    fatal_risks.append({
                        "id": "PLACEBO_HARDCODED_FIGURES",
                        "track": TRACK_FATAL,
                        "module": "no_placebo",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Dashboard or route displays hardcoded sample revenue/orders",
                        "title_id": "Angka omzet atau data pesanan masih teks karangan mati (hardcoded)",
                        "business_impact_en": "Numbers shown to you on screen are static sample figures, not real live business transactions.",
                        "business_impact_id": "Angka yang tampil di layar Anda adalah angka contoh buatan AI, bukan hasil hitungan transaksi asli dari database.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Query real aggregated rows from the database instead of returning hardcoded figures or mock arrays in {rel}.",
                    })

                # 2. Cash-Register
                if (
                    RE_SQL_FLOAT_MONEY.search(raw_line)
                    or RE_GO_FLOAT_MONEY.search(raw_line)
                    or RE_JS_FLOAT_MONEY.search(raw_line)
                    or RE_PY_FLOAT_MONEY.search(raw_line)
                ):
                    fatal_risks.append({
                        "id": "MONEY_FLOAT_ROUNDING",
                        "track": TRACK_FATAL,
                        "module": "cash_register",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Money calculation uses imprecise decimal floating-point numbers",
                        "title_id": "Hitungan uang memakai tipe angka desimal (float) yang rawan selisih",
                        "business_impact_en": "Invoices, discounts, taxes, or balances can silently lose or add cents on repeated transactions.",
                        "business_impact_id": "Tagihan, diskon, pajak, atau saldo bisa meleset beberapa perak/rupiah saat transaksi dijumlahkan.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Store and compute all monetary values as integer minor units (int64 cents) or exact SQL NUMERIC/DECIMAL in {rel}.",
                    })
                if RE_CLIENT_PRICE.search(raw_line):
                    fatal_risks.append({
                        "id": "MONEY_CLIENT_PRICE_TRUST",
                        "track": TRACK_FATAL,
                        "module": "cash_register",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Checkout trusts item price sent directly from the buyer's browser",
                        "title_id": "Kasir mempercayai harga barang yang dikirim dari browser pembeli",
                        "business_impact_en": "A buyer can edit browser request data and purchase expensive items for $0.01.",
                        "business_impact_id": "Pembeli nakal bisa mengubah harga barang dari browser menjadi Rp 1 saat checkout.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Never trust client-submitted price fields in {rel}; look up authoritative unit prices from the server database by item SKU/ID.",
                    })

                # 3. Door-Locks
                if RE_PUBLIC_SECRET.search(raw_line) or RE_LIVE_KEY.search(raw_line):
                    fatal_risks.append({
                        "id": "DOOR_EXPOSED_SECRET",
                        "track": TRACK_FATAL,
                        "module": "door_locks",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Secret API or database key is exposed in code or browser config",
                        "title_id": "Kunci rahasia API atau database terpampang terbuka di kode",
                        "business_impact_en": "Visitors can inspect your website bundle or repo, steal this key, and drain your account or database.",
                        "business_impact_id": "Orang luar bisa mengambil kunci rahasia ini dari website untuk menguras kuota API atau isi database Anda.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Move the secret key out of {rel} into a server-only environment variable (never NEXT_PUBLIC_ or VITE_) and rotate the exposed credential.",
                    })
                if RE_RAW_SQL.search(raw_line):
                    fatal_risks.append({
                        "id": "DOOR_RAW_SQL_INJECTION",
                        "track": TRACK_FATAL,
                        "module": "door_locks",
                        "file": rel,
                        "line": line_no,
                        "title_en": "Database query builds SQL directly from raw text input",
                        "title_id": "Pencarian database menyambung teks mentah yang rawan dibobol",
                        "business_impact_en": "An attacker can type crafted input into a form to dump or wipe your database tables.",
                        "business_impact_id": "Penyerang bisa memasukkan teks khusus pada form untuk menyedot atau menghapus isi tabel database.",
                        "evidence_snippet": trimmed[:140],
                        "repair_instruction": f"Replace string-concatenated SQL in {rel} with parameterized placeholders ($1, ?) and bound arguments.",
                    })

                # 4. Customer-Ready Polish Track
                if ext in {".tsx", ".jsx", ".html", ".ts", ".js", ".go", ".py"}:
                    if RE_FAKE_CONTACT.search(raw_line) or RE_FAKE_SOCIAL.search(raw_line):
                        polish_warnings.append({
                            "id": "POLISH_FAKE_PLACEHOLDER_TRUST",
                            "track": TRACK_POLISH,
                            "module": "customer_ready",
                            "file": rel,
                            "line": line_no,
                            "title_en": "AI sample contact details or unverified customer-count claim left on screen",
                            "title_id": "Kontak contoh (example.com / nomor palsu) atau klaim jumlah pelanggan karangan AI masih tertinggal",
                            "business_impact_en": "Real visitors will immediately spot fake contact info or made-up '10,000+ customers' claims and lose trust in your brand.",
                            "business_impact_id": "Calon pembeli asli akan langsung sadar ada email/nomor telepon contoh atau klaim angka karangan AI sehingga ragu bertransaksi.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Replace placeholder contacts/claims in {rel} with verified business details from the owner or Axoloom vault.",
                        })
                    if RE_DEAD_ANCHOR.search(raw_line):
                        polish_warnings.append({
                            "id": "POLISH_DEAD_LINK",
                            "track": TRACK_POLISH,
                            "module": "customer_ready",
                            "file": rel,
                            "line": line_no,
                            "title_en": "Navigation or footer link points nowhere (href='#')",
                            "title_id": "Tautan menu atau tombol mengarah ke jalan buntu (href='#')",
                            "business_impact_en": "Clicking this link leaves customers stuck on the same page, making the site feel unfinished.",
                            "business_impact_id": "Saat diklik pelanggan, tautan ini buntu dan membuat website terasa belum selesai dikerjakan.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Point the anchor tag in {rel} to a real destination URL/route, or change it into a <button type=\"button\"> with a real handler.",
                        })
                    if RE_RAW_STACK.search(raw_line):
                        polish_warnings.append({
                            "id": "POLISH_RAW_ERROR_EXPOSED",
                            "track": TRACK_POLISH,
                            "module": "customer_ready",
                            "file": rel,
                            "line": line_no,
                            "title_en": "Raw technical error trace can leak onto customer screens",
                            "title_id": "Pesan error koding mentah bisa bocor ke layar pelanggan",
                            "business_impact_en": "When something fails, customers see scary internal code traces instead of a helpful human message.",
                            "business_impact_id": "Saat terjadi kendala, pelanggan melihat teks koding internal yang membingungkan, bukan panduan bahasa manusia.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Log internal stack traces on the server only and return a clear, polite human error message in {rel}.",
                        })
                    if RE_ROBOTIC_COPY.search(raw_line):
                        polish_warnings.append({
                            "id": "POLISH_ROBOTIC_AI_PROSE",
                            "track": TRACK_POLISH,
                            "module": "customer_ready",
                            "file": rel,
                            "line": line_no,
                            "title_en": "Interface text uses generic AI brochure cliches",
                            "title_id": "Kalimat tampilan memakai bahasa brosur robotik khas AI",
                            "business_impact_en": "Generic AI hype phrases make your business sound automated rather than written by real people.",
                            "business_impact_id": "Kalimat klise khas robot mengurangi kesan tepercaya dan manusiawi dari bisnis Anda.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Rewrite the copy in {rel} using plain, concrete words stating exactly what the customer gets.",
                        })
                    if RE_TINY_BUTTON.search(raw_line):
                        polish_warnings.append({
                            "id": "POLISH_TINY_MOBILE_BUTTON",
                            "track": TRACK_POLISH,
                            "module": "customer_ready",
                            "file": rel,
                            "line": line_no,
                            "title_en": "Button is too small to tap comfortably on a phone screen (<44px)",
                            "title_id": "Ukuran tombol terlalu kecil untuk dipencet jempol di layar HP (<44px)",
                            "business_impact_en": "Mobile customers may mis-tap or struggle to press this control.",
                            "business_impact_id": "Pelanggan yang membuka dari HP akan kesulitan memencet tombol ini dengan jempol.",
                            "evidence_snippet": trimmed[:140],
                            "repair_instruction": f"Increase interactive button dimensions/padding in {rel} to at least 44x44px.",
                        })

    return {
        "files_scanned": files_scanned,
        "fatal_risks": deduplicate_findings(fatal_risks),
        "polish_warnings": deduplicate_findings(polish_warnings),
        "discovered_urls": discovered_urls,
    }


def deduplicate_findings(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    if len(items) <= 1:
        return items
    idx_by_key: dict[tuple[str, str], int] = {}
    counts: dict[tuple[str, str], int] = {}
    out: list[dict[str, Any]] = []
    for it in items:
        key = (str(it.get("id", "")), str(it.get("file", "")))
        counts[key] = counts.get(key, 0) + 1
        if key in idx_by_key:
            base_instr = it["repair_instruction"]
            out[idx_by_key[key]]["repair_instruction"] = f"{base_instr} (occurs {counts[key]} times in {key[1]})"
            continue
        idx_by_key[key] = len(out)
        out.append(dict(it))
    return out


# ---------------------------------------------------------------------------
# Two-Way Axoloom SQLite Graph Bridge (.axoloom/index.db)
# ---------------------------------------------------------------------------

def locate_axoloom_db(work_dir: Path) -> tuple[Path | None, Path | None]:
    cur = work_dir.resolve()
    for _ in range(3):
        for db_name in ("index.db", "axoloom.db"):
            cand = cur / ".axoloom" / db_name
            if cand.is_file():
                return cand, cur
        if cur.parent == cur:
            break
        cur = cur.parent
    return None, None


def inspect_axoloom_bridge(work_dir: Path, declared_protected: list[dict[str, Any]]) -> dict[str, Any]:
    db_path, axo_root = locate_axoloom_db(work_dir)
    base_ctx: dict[str, Any] = {
        "connected": False,
        "db_path": "",
        "indexed_endpoints": 0,
        "indexed_sql_tables": 0,
        "active_guardrails": 0,
        "active_directives": 0,
        "undeclared_routes_found": [],
        "writeback_path": "",
    }
    if not db_path or not axo_root:
        return base_ctx

    abs_work = work_dir.resolve()
    proj = abs_work.name
    path_like = str(abs_work) + "%"
    declared_set = {str(p.get("path", "")).strip() for p in declared_protected}

    try:
        uri = f"file:{db_path.as_posix()}?mode=ro"
        conn = sqlite3.connect(uri, uri=True, timeout=5.0)
        cur = conn.cursor()
        base_ctx["connected"] = True
        base_ctx["db_path"] = db_path.relative_to(axo_root).as_posix()

        try:
            if abs_work == axo_root:
                base_ctx["indexed_sql_tables"] = cur.execute("SELECT COUNT(*) FROM nodes WHERE kind = 'table'").fetchone()[0]
                base_ctx["active_guardrails"] = cur.execute("SELECT COUNT(*) FROM guardrails").fetchone()[0]
                ep_rows = cur.execute("SELECT name FROM nodes WHERE kind IN ('endpoint', 'route')").fetchall()
            else:
                base_ctx["indexed_sql_tables"] = cur.execute(
                    "SELECT COUNT(*) FROM nodes WHERE kind = 'table' AND (LOWER(project) = LOWER(?) OR path LIKE ?)",
                    (proj, path_like),
                ).fetchone()[0]
                base_ctx["active_guardrails"] = cur.execute(
                    "SELECT COUNT(*) FROM guardrails WHERE LOWER(project) = LOWER(?) OR source_path LIKE ?",
                    (proj, path_like),
                ).fetchone()[0]
                ep_rows = cur.execute(
                    "SELECT name FROM nodes WHERE kind IN ('endpoint', 'route') AND (LOWER(project) = LOWER(?) OR path LIKE ?)",
                    (proj, path_like),
                ).fetchall()
        except sqlite3.Error:
            ep_rows = []

        try:
            base_ctx["active_directives"] = cur.execute("SELECT COUNT(*) FROM directives").fetchone()[0]
        except sqlite3.Error:
            pass

        seen: set[str] = set()
        undeclared: list[str] = []
        for (ep_name,) in ep_rows:
            route = ""
            for tok in str(ep_name).split():
                if tok.startswith("/"):
                    route = tok
                    break
            if not route and str(ep_name).startswith("/"):
                route = str(ep_name)
            if not route or route in seen:
                continue
            seen.add(route)
            base_ctx["indexed_endpoints"] += 1
            if RE_SENSITIVE_ROUTE.search(route) and route not in declared_set:
                undeclared.append(route)
        base_ctx["undeclared_routes_found"] = undeclared
        conn.close()
    except sqlite3.Error:
        pass

    return base_ctx


def expand_blast_radius_files(work_dir: Path, rel_files: set[str]) -> set[str]:
    if not rel_files:
        return rel_files
    db_path, _ = locate_axoloom_db(work_dir)
    if not db_path:
        return rel_files
    abs_work = work_dir.resolve()
    expanded = set(rel_files)
    try:
        uri = f"file:{db_path.as_posix()}?mode=ro"
        conn = sqlite3.connect(uri, uri=True, timeout=5.0)
        cur = conn.cursor()
        for rel in list(rel_files):
            abs_target = str(abs_work / rel)
            rows = cur.execute(
                """
                SELECT DISTINCT e.path
                FROM nodes n
                JOIN edges e ON (e.dst_id = n.id OR e.dst_name = n.name)
                WHERE n.path = ? OR n.path LIKE ?
                """,
                (abs_target, "%/" + rel),
            ).fetchall()
            for (caller_path,) in rows:
                if not caller_path:
                    continue
                cp = Path(str(caller_path))
                try:
                    c_rel = cp.resolve().relative_to(abs_work).as_posix()
                    expanded.add(c_rel)
                except ValueError:
                    if not cp.is_absolute():
                        expanded.add(cp.as_posix())
        conn.close()
    except sqlite3.Error:
        pass
    return expanded


def sync_back_to_axoloom(work_dir: Path, receipt: dict[str, Any]) -> str:
    _, axo_root = locate_axoloom_db(work_dir)
    if not axo_root:
        return ""
    axo_bin = shutil.which("axoloom")
    if not axo_bin:
        cand = Path.home() / ".local" / "bin" / ("axoloom.exe" if os.name == "nt" else "axoloom")
        if cand.exists():
            axo_bin = str(cand)
    if not axo_bin:
        return ""
    summary = (
        f"AxoProof [{receipt['verdict']}] on {receipt['project_name']}: "
        f"{len(receipt['fatal_risks'])} fatal risks, {len(receipt['polish_warnings'])} polish items, "
        f"{len(receipt['runtime_probes'])} runtime probes (seal {receipt['seal_sha256'][:12]})"
    )
    try:
        proc = subprocess.run([axo_bin, "handoff", "--dir", str(axo_root), "axoproof", summary], capture_output=True, timeout=5, check=False)
        if proc.returncode == 0:
            return ".axoloom/handoff/axoproof.md"
    except Exception:
        pass
    return ""


# ---------------------------------------------------------------------------
# Counterfactual Live Runtime Probes + Auto-Start + Headless 375px DOM Probe
# ---------------------------------------------------------------------------

class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def http_request(method: str, url: str, headers: dict[str, str] | None = None, body: str | None = None, timeout: float = 4.0) -> tuple[int, str]:
    opener = urllib.request.build_opener(_NoRedirect)
    data = body.encode("utf-8") if body is not None else None
    req = urllib.request.Request(url=url, data=data, method=method.upper())
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    try:
        with opener.open(req, timeout=timeout) as resp:
            raw = resp.read(262144).decode("utf-8", errors="ignore")
            return resp.status, raw
    except urllib.error.HTTPError as e:
        try:
            raw = e.read(65536).decode("utf-8", errors="ignore")
        except Exception:
            raw = ""
        return e.code, raw


def validate_loopback_target(raw_url: str, allow_remote: bool) -> str | None:
    parsed = urllib.parse.urlparse(raw_url)
    if parsed.scheme not in {"http", "https"}:
        return f"unsupported scheme {parsed.scheme!r}"
    if allow_remote:
        return None
    host = (parsed.hostname or "").lower()
    if host == "localhost":
        return None
    try:
        if ipaddress.ip_address(host).is_loopback:
            return None
    except ValueError:
        pass
    return f"host {host!r} is not loopback (127.0.0.1/localhost)"


def start_background_server(work_dir: Path, command_line: str) -> tuple[subprocess.Popen[bytes] | None, Any]:
    if os.name == "nt":
        proc = subprocess.Popen(
            ["cmd.exe", "/C", command_line],
            cwd=str(work_dir),
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        def stop_win() -> None:
            subprocess.run(["taskkill", "/T", "/F", "/PID", str(proc.pid)], capture_output=True, check=False)
            try:
                proc.kill()
            except Exception:
                pass
        return proc, stop_win
    else:
        proc = subprocess.Popen(
            ["sh", "-c", command_line],
            cwd=str(work_dir),
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            start_new_session=True,
        )
        def stop_unix() -> None:
            try:
                os.killpg(proc.pid, signal.SIGTERM)
            except ProcessLookupError:
                return
            try:
                proc.wait(timeout=1.5)
            except subprocess.TimeoutExpired:
                try:
                    os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
        return proc, stop_unix


def count_sqlite_rows(db_path: Path, table: str) -> int:
    if not VALID_SQL_IDENT.match(table):
        raise ValueError(f"unsafe SQLite table identifier {table!r}")
    if not db_path.is_file():
        raise FileNotFoundError(f"sqlite file not found at {db_path}")
    conn = sqlite3.connect(str(db_path), timeout=5.0)
    try:
        row = conn.execute(f'SELECT COUNT(*) FROM "{table}"').fetchone()
        return int(row[0]) if row else 0
    finally:
        conn.close()


def count_json_items(raw: str) -> int:
    try:
        parsed = json.loads(raw)
    except Exception:
        return 0
    if isinstance(parsed, list):
        return len(parsed)
    if isinstance(parsed, dict):
        for key in ("items", "data", "orders", "records", "results", "rows"):
            if isinstance(parsed.get(key), list):
                return len(parsed[key])
        return len(parsed)
    return 0


def detect_headless_browser() -> str:
    env_bin = os.environ.get("AXOPROOF_BROWSER_BIN", "").strip()
    if env_bin and Path(env_bin).exists():
        return env_bin
    sys_name = platform.system().lower()
    candidates: list[str] = []
    if sys_name == "darwin":
        candidates = [
            "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
            "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
            "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
            "/Applications/Chromium.app/Contents/MacOS/Chromium",
        ]
    elif sys_name == "windows":
        candidates = [
            r"C:\Program Files\Google\Chrome\Application\chrome.exe",
            r"C:\Program Files (x86)\Google\Chrome\Application\chrome.exe",
            r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe",
            r"C:\Program Files\Microsoft\Edge\Application\msedge.exe",
        ]
    else:
        for name in ("google-chrome", "chromium", "chromium-browser", "microsoft-edge"):
            found = shutil.which(name)
            if found:
                return found
    for c in candidates:
        if Path(c).exists():
            return c
    return ""


def run_all_probes(work_dir: Path, claims: dict[str, Any], extra_sensitive_routes: list[str], allow_remote: bool) -> dict[str, Any]:
    out: dict[str, Any] = {
        "results": [],
        "fatal_findings": [],
        "all_executed": False,
        "any_executed": False,
    }
    base = str(claims.get("base_url", "")).strip().rstrip("/")
    if not base:
        return out

    err_msg = validate_loopback_target(base, allow_remote)
    if err_msg:
        out["results"].append({
            "name": "Target safety boundary check",
            "kind": "health_pulse",
            "executed": False,
            "passed": False,
            "target": base,
            "summary_en": "Skipped live probes: BaseURL must be localhost/127.0.0.1 unless --allow-remote-probe is set.",
            "summary_id": "Uji jalan dilewati: BaseURL wajib localhost/127.0.0.1 kecuali opsi --allow-remote-probe diaktifkan.",
            "raw_evidence": err_msg,
        })
        return out

    health_path = str(claims.get("health_path", "")).strip() or "/"
    if not health_path.startswith("/"):
        health_path = "/" + health_path

    stop_fn = None
    auto_started = False
    status_code = 0
    reachable = False

    try:
        try:
            status_code, _ = http_request("GET", base + health_path)
            reachable = True
        except Exception as first_err:
            start_cmd = str(claims.get("start_command", "")).strip()
            if start_cmd:
                _, stop_fn = start_background_server(work_dir, start_cmd)
                auto_started = True
                deadline = time.monotonic() + 12.0
                while time.monotonic() < deadline:
                    time.sleep(0.15)
                    try:
                        status_code, _ = http_request("GET", base + health_path)
                        reachable = True
                        break
                    except Exception:
                        continue
            if not reachable:
                out["results"].append({
                    "name": "Local application server pulse",
                    "kind": "health_pulse",
                    "executed": False,
                    "passed": False,
                    "target": base + health_path,
                    "summary_en": "Application server was not running locally during inspection; runtime behavior remains unproven.",
                    "summary_id": "Server aplikasi belum menyala saat diperiksa; perilaku jalan nyata belum teruji.",
                    "raw_evidence": str(first_err),
                })
                return out

        out["any_executed"] = True
        out["all_executed"] = True
        boot_en = " (auto-started via start_command)" if auto_started else ""
        boot_id = " (dinyalakan otomatis lewat start_command)" if auto_started else ""
        out["results"].append({
            "name": "Local application server pulse",
            "kind": "health_pulse",
            "executed": True,
            "passed": status_code < 500,
            "target": base + health_path,
            "http_status_got": status_code,
            "summary_en": f"Server responded live on {health_path} (HTTP {status_code}){boot_en}.",
            "summary_id": f"Server merespons aktif pada {health_path} (HTTP {status_code}){boot_id}.",
            "raw_evidence": f"GET {base + health_path} -> {status_code}",
        })

        # 2. Persistence Probes (SQLite, HTTP Read-Back for Postgres/Supabase/MySQL, or both)
        for pc in claims.get("persistence", []):
            res, finding = _probe_persistence(work_dir, base, pc)
            out["results"].append(res)
            if not res["executed"]:
                out["all_executed"] = False
            if finding:
                out["fatal_findings"].append(finding)

        # 3. Stranger Door Probes (Declared + Axoloom Auto-Discovered)
        route_map: dict[str, dict[str, str]] = {}
        for pr in claims.get("protected_routes", []):
            m = str(pr.get("method", "GET")).upper() or "GET"
            p = str(pr.get("path", "")).strip()
            route_map[f"{m} {p}"] = {"name": str(pr.get("name", p)), "method": m, "path": p}
        for extra in extra_sensitive_routes:
            key = f"GET {extra}"
            if key not in route_map:
                route_map[key] = {
                    "name": f"Auto-discovered sensitive endpoint ({extra})",
                    "method": "GET",
                    "path": extra,
                }
        for pr in route_map.values():
            res, finding = _probe_stranger_door(base, pr)
            out["results"].append(res)
            if finding:
                out["fatal_findings"].append(finding)

        # 4. Negative Quantity / Price Tamper Probes
        for mt in claims.get("money_tampers", []):
            res, finding = _probe_money_tamper(work_dir, base, mt)
            out["results"].append(res)
            if finding:
                out["fatal_findings"].append(finding)

        # 5. Headless 375px Mobile Screen DOM Probes
        screen_routes = [str(r) for r in claims.get("screen_routes", []) if str(r).strip()]
        if screen_routes:
            s_res, s_findings = _probe_headless_screens(base, screen_routes)
            out["results"].extend(s_res)
            out["fatal_findings"].extend(s_findings)

        return out
    finally:
        if stop_fn is not None:
            stop_fn()


def _probe_persistence(work_dir: Path, base: str, pc: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any] | None]:
    name = str(pc.get("name", "Persistence check"))
    method = str(pc.get("method", "POST")).upper() or "POST"
    path = "/" + str(pc.get("path", "")).lstrip("/")
    target_url = base + path
    sqlite_rel = str(pc.get("sqlite_path", "")).strip()
    table = str(pc.get("table", "")).strip()
    read_back = str(pc.get("read_back_path", "")).strip()
    has_sqlite = bool(sqlite_rel and table)
    has_rb = bool(read_back)

    if not has_sqlite and not has_rb:
        return {
            "name": name, "kind": "db_persistence", "executed": False, "passed": False,
            "target": target_url,
            "summary_en": "Persistence claim requires either sqlite_path+table or read_back_path.",
            "summary_id": "Klaim simpan data membutuhkan sqlite_path+table atau read_back_path.",
            "raw_evidence": "missing sqlite_path and read_back_path",
        }, None

    rows_before = 0
    db_full = (work_dir / sqlite_rel) if not Path(sqlite_rel).is_absolute() else Path(sqlite_rel)
    if has_sqlite:
        try:
            rows_before = count_sqlite_rows(db_full, table)
        except Exception as e:
            return {
                "name": name, "kind": "db_persistence", "executed": False, "passed": False,
                "target": target_url,
                "summary_en": f"Could not inspect SQLite table {table} before action: {e}",
                "summary_id": f"Gagal membaca tabel database {table} sebelum aksi: {e}",
                "raw_evidence": str(e),
            }, None

    rb_before_body = ""
    rb_before_cnt = 0
    rb_path_norm = "/" + read_back.lstrip("/") if has_rb else ""
    if has_rb:
        try:
            rb_status, rb_before_body = http_request("GET", base + rb_path_norm, pc.get("read_headers"))
            if rb_status >= 500:
                raise RuntimeError(f"read-back HTTP {rb_status}")
            rb_before_cnt = count_json_items(rb_before_body)
        except Exception as e:
            return {
                "name": name, "kind": "db_persistence", "executed": False, "passed": False,
                "target": target_url,
                "summary_en": "Could not read baseline state from read_back_path before action.",
                "summary_id": "Gagal membaca data awal dari read_back_path sebelum aksi.",
                "raw_evidence": str(e),
            }, None

    hdrs = {"Content-Type": "application/json"}
    hdrs.update(pc.get("headers") or {})
    try:
        code, _ = http_request(method, target_url, hdrs, str(pc.get("payload_json", "{}")))
    except Exception as e:
        return {
            "name": name, "kind": "db_persistence", "executed": False, "passed": False,
            "target": target_url,
            "summary_en": "Request failed to reach local endpoint.",
            "summary_id": "Permintaan gagal menjangkau endpoint lokal.",
            "raw_evidence": str(e),
        }, None

    max_status = int(pc.get("expect_status_max") or 299)
    min_delta = max(1, int(pc.get("expect_min_delta") or 1))
    status_ok = 200 <= code <= max_status

    rows_after = rows_before
    sqlite_ok = True
    if has_sqlite:
        try:
            rows_after = count_sqlite_rows(db_full, table)
        except Exception:
            rows_after = rows_before
        sqlite_ok = (rows_after - rows_before) >= min_delta

    rb_ok = True
    rb_after_cnt = rb_before_cnt
    rb_ev = ""
    if has_rb:
        try:
            rb_code, rb_after_body = http_request("GET", base + rb_path_norm, pc.get("read_headers"))
            if rb_code >= 500:
                rb_ok = False
                rb_ev = f"read-back HTTP {rb_code}"
            else:
                rb_after_cnt = count_json_items(rb_after_body)
                delta_rb = rb_after_cnt - rb_before_cnt
                marker = str(pc.get("expect_read_substring", "")).strip()
                if marker:
                    had_before = marker in rb_before_body
                    has_after = marker in rb_after_body
                    rb_ok = has_after and (not had_before or delta_rb >= min_delta)
                    rb_ev = f"GET {rb_path_norm} items: {rb_before_cnt} -> {rb_after_cnt} (marker {marker}={has_after})"
                else:
                    rb_ok = delta_rb >= min_delta
                    rb_ev = f"GET {rb_path_norm} items: {rb_before_cnt} -> {rb_after_cnt} (delta {delta_rb:+d})"
        except Exception:
            rb_ok = False
            rb_ev = "read-back failed"

    passed = status_ok and sqlite_ok and rb_ok
    parts = [f"HTTP {code}"]
    if has_sqlite:
        parts.append(f"{table} rows: {rows_before} -> {rows_after} (delta {rows_after - rows_before:+d})")
    if has_rb:
        parts.append(rb_ev)
    raw_ev = " | ".join(parts)
    store_label = table or rb_path_norm

    res = {
        "name": name,
        "kind": "db_persistence",
        "executed": True,
        "passed": passed,
        "target": f"{method} {path} -> {store_label}",
        "http_status_got": code,
        "rows_before": rows_before,
        "rows_after": rows_after,
        "raw_evidence": raw_ev,
    }
    if passed:
        if has_sqlite:
            res["summary_en"] = f"Proven real save: table {table} grew from {rows_before} to {rows_after} rows (HTTP {code})."
            res["summary_id"] = f"Terbukti tersimpan nyata: isi tabel {table} bertambah dari {rows_before} menjadi {rows_after} baris (HTTP {code})."
        else:
            res["summary_en"] = f"Proven persistent save via read-back on {rb_path_norm} (items {rb_before_cnt} -> {rb_after_cnt}, HTTP {code})."
            res["summary_id"] = f"Terbukti tersimpan permanen saat dibaca ulang di {rb_path_norm} (data {rb_before_cnt} menjadi {rb_after_cnt}, HTTP {code})."
        return res, None

    res["summary_en"] = f"FAILED persistence proof on {store_label} ({raw_ev})."
    res["summary_id"] = f"GAGAL bukti simpan permanen pada {store_label} ({raw_ev})."
    finding = {
        "id": "RUNTIME_GHOST_PERSISTENCE",
        "track": TRACK_FATAL,
        "module": "live_persistence",
        "title_en": f"Action ({name}) did not persist durable data to {store_label}",
        "title_id": f"Fitur ({name}) tidak benar-benar menyimpan data permanen ke {store_label}",
        "business_impact_en": f"Submitting the action returned HTTP {code}, yet durable storage check failed ({raw_ev}). Customer orders or records will vanish.",
        "business_impact_id": f"Saat dikirim server menjawab HTTP {code}, namun pemeriksaan simpan data gagal ({raw_ev}). Pesanan atau isian pelanggan akan hilang.",
        "evidence_snippet": raw_ev,
        "repair_instruction": f"Fix {method} {path} so it commits durable records to your database ({store_label}) before returning 2xx.",
    }
    return res, finding


def _probe_stranger_door(base: str, pr: dict[str, str]) -> tuple[dict[str, Any], dict[str, Any] | None]:
    name = pr["name"]
    method = pr["method"]
    path = "/" + pr["path"].lstrip("/")
    try:
        code, _ = http_request(method, base + path)
    except Exception as e:
        return {
            "name": name, "kind": "stranger_door", "executed": False, "passed": False,
            "target": base + path,
            "summary_en": "Could not reach protected route for unauthenticated test.",
            "summary_id": "Gagal menghubungi endpoint privat untuk uji akses tanpa login.",
            "raw_evidence": str(e),
        }, None

    locked = code in (401, 403) or (name.startswith("Auto-discovered") and code == 405)
    res = {
        "name": name,
        "kind": "stranger_door",
        "executed": True,
        "passed": locked,
        "target": f"{method} {path} (no login)",
        "http_status_got": code,
        "raw_evidence": f"Unauthenticated {method} {path} -> HTTP {code}",
    }
    if locked:
        res["summary_en"] = f"Door locked: unauthenticated stranger blocked with HTTP {code} on {path}."
        res["summary_id"] = f"Pintu terkunci rapat: akses orang asing tanpa login ditolak dengan HTTP {code} pada {path}."
        return res, None

    res["summary_en"] = f"UNLOCKED DOOR: unauthenticated request to {path} returned HTTP {code} instead of 401/403!"
    res["summary_id"] = f"PINTU TERBUKA TANPA LOGIN: akses tanpa login ke {path} membalas HTTP {code} (seharusnya ditolak 401/403)!"
    finding = {
        "id": "RUNTIME_UNLOCKED_DOOR",
        "track": TRACK_FATAL,
        "module": "door_locks",
        "title_en": f"Private endpoint {path} opened for an unlogged stranger (HTTP {code})",
        "title_id": f"Halaman/API privat {path} bisa dibuka orang asing tanpa login (HTTP {code})",
        "business_impact_en": f"AxoProof knocked on {path} without logging in and received HTTP {code}. Customer or business records are exposed.",
        "business_impact_id": f"AxoProof mencoba masuk ke {path} tanpa login dan justru diberi akses HTTP {code}. Data pelanggan atau bisnis bisa bocor ke publik.",
        "evidence_snippet": res["raw_evidence"],
        "repair_instruction": f"Enforce authentication check returning HTTP 401/403 for unauthenticated requests on {method} {path}.",
    }
    return res, finding


def _probe_money_tamper(work_dir: Path, base: str, mt: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any] | None]:
    name = str(mt.get("name", "Money tamper check"))
    method = str(mt.get("method", "POST")).upper() or "POST"
    path = "/" + str(mt.get("path", "")).lstrip("/")
    sqlite_rel = str(mt.get("sqlite_path", "")).strip()
    table = str(mt.get("table", "")).strip()
    has_db = bool(sqlite_rel and table)
    db_full = (work_dir / sqlite_rel) if not Path(sqlite_rel).is_absolute() else Path(sqlite_rel)

    rows_before = 0
    if has_db:
        try:
            rows_before = count_sqlite_rows(db_full, table)
        except Exception:
            rows_before = 0

    hdrs = {"Content-Type": "application/json"}
    hdrs.update(mt.get("headers") or {})
    try:
        code, _ = http_request(method, base + path, hdrs, str(mt.get("tampered_body_json", "{}")))
    except Exception as e:
        return {
            "name": name, "kind": "negative_tamper", "executed": False, "passed": False,
            "target": base + path,
            "summary_en": "Could not reach transaction endpoint for tamper probe.",
            "summary_id": "Gagal menghubungi endpoint transaksi untuk uji manipulasi angka.",
            "raw_evidence": str(e),
        }, None

    rows_after = rows_before
    if has_db:
        try:
            rows_after = count_sqlite_rows(db_full, table)
        except Exception:
            rows_after = rows_before

    rejected = 400 <= code < 500 and rows_after == rows_before
    res = {
        "name": name,
        "kind": "negative_tamper",
        "executed": True,
        "passed": rejected,
        "target": f"{method} {path} (tampered payload)",
        "http_status_got": code,
        "rows_before": rows_before,
        "rows_after": rows_after,
        "raw_evidence": f"Tamper {path} -> HTTP {code} (rows {rows_before} -> {rows_after})",
    }
    if rejected:
        res["summary_en"] = f"Cash register defended: tampered quantity/price rejected with HTTP {code} and 0 rows written."
        res["summary_id"] = f"Kasir aman: percobaan manipulasi harga/jumlah minus ditolak dengan HTTP {code} dan 0 data masuk."
        return res, None

    res["summary_en"] = f"CASH REGISTER EXPLOITED: tampered payload accepted with HTTP {code}!"
    res["summary_id"] = f"CELAH KASIR TEMBUS: input jumlah minus atau manipulasi harga lolos dengan HTTP {code}!"
    finding = {
        "id": "RUNTIME_MONEY_TAMPER_ACCEPTED",
        "track": TRACK_FATAL,
        "module": "cash_register",
        "title_en": f"Checkout accepted negative quantity or tampered price on {path} (HTTP {code})",
        "title_id": f"Kasir meloloskan pesanan jumlah minus atau harga dimanipulasi pada {path} (HTTP {code})",
        "business_impact_en": f"Sending negative quantity or tampered price to {path} succeeded with HTTP {code}. Buyers can manipulate totals or trigger negative charges.",
        "business_impact_id": f"Pengiriman jumlah minus atau manipulasi harga ke {path} justru diterima dengan HTTP {code}. Pembeli nakal bisa mengakali total bayar.",
        "evidence_snippet": res["raw_evidence"],
        "repair_instruction": f"Validate quantity > 0, ignore client-supplied price fields, and return HTTP 400/422 on invalid inputs in {method} {path}.",
    }
    return res, finding


def _probe_headless_screens(base: str, routes: list[str]) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    browser = detect_headless_browser()
    if not browser:
        return [{
            "name": "Headless browser screen render check",
            "kind": "screen_dom",
            "executed": False,
            "passed": False,
            "target": ", ".join(routes),
            "summary_en": "Skipped headless screen render check: no local Chrome/Edge/Chromium binary detected.",
            "summary_id": "Pemeriksaan layar browser dilewati: Chrome/Edge/Chromium tidak ditemukan di perangkat ini.",
            "raw_evidence": "browser binary not installed",
        }], []

    results: list[dict[str, Any]] = []
    findings: list[dict[str, Any]] = []
    for raw_r in routes:
        route = "/" + raw_r.strip().lstrip("/")
        url = base + route
        try:
            proc = subprocess.run(
                [
                    browser, "--headless=new", "--disable-gpu", "--no-first-run",
                    "--no-default-browser-check", "--window-size=375,812",
                    "--virtual-time-budget=2000", "--dump-dom", url,
                ],
                capture_output=True, text=True, timeout=8, check=False,
            )
            dom = proc.stdout
        except Exception as e:
            results.append({
                "name": f"Mobile 375px screen render ({route})",
                "kind": "screen_dom",
                "executed": False,
                "passed": False,
                "target": url,
                "summary_en": f"Headless browser could not render {route} within timeout.",
                "summary_id": f"Browser headless gagal membuka {route} dalam batas waktu.",
                "raw_evidence": str(e),
            })
            continue

        visible_words = RE_STRIP_HTML.sub(" ", dom).split()
        crash_hit = RE_CRASH_OVERLAY.search(dom)
        if crash_hit or len(visible_words) == 0:
            reason = crash_hit.group(0) if crash_hit else "blank empty screen (0 visible words)"
            results.append({
                "name": f"Mobile 375px screen render ({route})",
                "kind": "screen_dom",
                "executed": True,
                "passed": False,
                "target": url,
                "summary_en": f"Screen {route} crashed or rendered completely blank ({reason}).",
                "summary_id": f"Halaman {route} tampil putih kosong atau error JavaScript ({reason}).",
                "raw_evidence": f"DOM crash/blank: {reason}",
            })
            findings.append({
                "id": "RUNTIME_SCREEN_CRASH_OR_BLANK",
                "track": TRACK_FATAL,
                "module": "no_placebo",
                "title_en": f"Page {route} renders a blank screen or client crash in browser",
                "title_id": f"Halaman {route} tampil putih kosong atau error saat dibuka di browser",
                "business_impact_en": f"Opening {route} in a real 375px phone browser resulted in {reason}.",
                "business_impact_id": f"Saat halaman {route} dibuka di browser ukuran HP 375px, layar kosong atau muncul error ({reason}).",
                "evidence_snippet": reason,
                "repair_instruction": f"Inspect client hydration/JS runtime errors on route {route} and ensure visible content renders cleanly.",
            })
            continue

        joined_vis = " ".join(visible_words)
        nan_hit = RE_NAN_CURRENCY.search(joined_vis)
        if nan_hit:
            bad_tok = nan_hit.group(0)
            results.append({
                "name": f"Mobile 375px screen render ({route})",
                "kind": "screen_dom",
                "executed": True,
                "passed": False,
                "target": url,
                "summary_en": f"Screen {route} displays broken number text ({bad_tok}) to customers.",
                "summary_id": f"Halaman {route} menampilkan angka rusak ({bad_tok}) ke layar pelanggan.",
                "raw_evidence": f"Rendered broken value: {bad_tok}",
            })
            findings.append({
                "id": "RUNTIME_SCREEN_BROKEN_NUMBER",
                "track": TRACK_FATAL,
                "module": "cash_register",
                "title_en": f"Page {route} displays broken NaN/undefined price or number on screen",
                "title_id": f"Halaman {route} menampilkan tulisan harga/angka rusak ({bad_tok}) di layar",
                "business_impact_en": f"Customers opening {route} see broken calculation output ({bad_tok}) on screen.",
                "business_impact_id": f"Pelanggan yang membuka halaman {route} melihat angka error ({bad_tok}) langsung di layar.",
                "evidence_snippet": bad_tok,
                "repair_instruction": f"Guard missing/null numeric fields and format currency safely on route {route}.",
            })
            continue

        results.append({
            "name": f"Mobile 375px screen render ({route})",
            "kind": "screen_dom",
            "executed": True,
            "passed": True,
            "target": url,
            "summary_en": f"Rendered cleanly in real headless browser ({len(visible_words)} visible words, zero JS crash or NaN).",
            "summary_id": f"Tampil utuh di browser asli ukuran HP ({len(visible_words)} kata terbaca, nol layar putih atau angka NaN).",
            "raw_evidence": f"375x812 DOM ok ({len(visible_words)} words)",
        })

    return results, findings


# ---------------------------------------------------------------------------
# Cryptographic Seal & Three-Surface Renderers (CLI Table, Markdown, Swiss HTML)
# ---------------------------------------------------------------------------

def compute_seal(receipt: dict[str, Any]) -> str:
    clone = dict(receipt)
    clone["seal_sha256"] = ""
    canonical = json.dumps(clone, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(canonical).hexdigest()


def verify_seal(receipt: dict[str, Any]) -> bool:
    seal = str(receipt.get("seal_sha256", ""))
    if len(seal) != 64:
        return False
    return compute_seal(receipt) == seal


def render_terminal(r: dict[str, Any], html_path: Path, lang: str, color: bool) -> str:
    is_id = lang.lower() == "id"
    use_ansi = color and not os.environ.get("NO_COLOR") and os.environ.get("TERM") != "dumb"
    bold = "\033[1m" if use_ansi else ""
    dim = "\033[2m" if use_ansi else ""
    green = "\033[32m" if use_ansi else ""
    amber = "\033[33m" if use_ansi else ""
    red = "\033[31m" if use_ansi else ""
    rst = "\033[0m" if use_ansi else ""

    hr = "─" * 74
    thin = "·" * 74
    verdict = r["verdict"]
    if verdict == VERDICT_SAFE:
        badge = "[AMAN DITES]" if is_id else "[VERIFIED]"
        scol = green
    elif verdict == VERDICT_HOLD:
        badge = "[TAHAN RILIS]" if is_id else "[HOLD RELEASE]"
        scol = red
    else:
        badge = "[CEK MANUAL]" if is_id else "[PENDING]"
        scol = amber

    headline = r["headline_id"] if is_id else r["headline_en"]
    title_lbl = "AxoProof Lembar Pemeriksaan CLI" if is_id else "AxoProof Inspection Ledger"
    axo = r["axoloom"]
    if axo.get("connected"):
        axo_txt = (
            f"Aktif ({axo['indexed_endpoints']} route, {axo['indexed_sql_tables']} tabel)"
            if is_id
            else f"Active ({axo['indexed_endpoints']} routes, {axo['indexed_sql_tables']} tables)"
        )
    else:
        axo_txt = "Mandiri" if is_id else "Standalone"

    lines = [
        hr,
        f"{bold}{title_lbl}{rst}  ({r['project_name']})",
        f"{scol}{bold}{badge} {headline}{rst}",
        hr,
    ]
    if is_id:
        lines.append(f"Berkas diperiksa : {r['files_scanned']:<6d} | Masalah fatal : {len(r['fatal_risks'])} | Catatan layar : {len(r['polish_warnings'])}")
        lines.append(f"Uji jalan live   : {len(r['runtime_probes']):<6d} | Otak Axoloom  : {axo_txt}")
    else:
        lines.append(f"Files inspected  : {r['files_scanned']:<6d} | Blocking risks : {len(r['fatal_risks'])} | Screen notes : {len(r['polish_warnings'])}")
        lines.append(f"Live probes      : {len(r['runtime_probes']):<6d} | Axoloom link   : {axo_txt}")

    lines.append(thin)
    lines.append(f"{bold}{'01. Kasir, Kunci Data & Fitur Asli' if is_id else '01. Money, Privacy & Real Wiring'}{rst}")
    if not r["fatal_risks"]:
        lines.append(
            f"  {green}[LULUS]{rst} Bersih dari tombol pajangan, salah tipe uang, dan pintu terbuka."
            if is_id
            else f"  {green}[PASS]{rst}  No fake buttons, float currency bugs, or unlocked routes."
        )
    else:
        for i, f in enumerate(r["fatal_risks"], start=1):
            t = f["title_id"] if is_id else f["title_en"]
            imp = f["business_impact_id"] if is_id else f["business_impact_en"]
            loc = f" [{f['file']}:{f['line']}]" if f.get("file") else ""
            lines.append(f"  {red}{bold}{i}. {t}{rst}{dim}{loc}{rst}")
            lines.append(f"     {imp}")

    lines.append(thin)
    lines.append(f"{bold}{'02. Pengukuran Server & Database Live' if is_id else '02. Live Server & Database Ledger'}{rst}")
    if not r["runtime_probes"]:
        lines.append(
            f"  {amber}[TUNDA]{rst} Server lokal belum dikonfigurasi pada .axoproof/claims.json."
            if is_id
            else f"  {amber}[SKIP]{rst}  No live BaseURL configured in .axoproof/claims.json yet."
        )
    else:
        for p in r["runtime_probes"]:
            if p["executed"] and p["passed"]:
                tag, col = ("[BUKTI]" if is_id else "[PROVEN]"), green
            elif p["executed"]:
                tag, col = ("[GAGAL]" if is_id else "[FAILED]"), red
            else:
                tag, col = "[SKIP]", amber
            nm = p["name"] if len(p["name"]) <= 36 else p["name"][:33] + "..."
            lines.append(f"  {col}{bold}{tag:<8s}{rst} {nm:<36s} {dim}{p['raw_evidence']}{rst}")

    if r["polish_warnings"]:
        lines.append(thin)
        hdr = f"03. Catatan Layar & Bahasa Manusia ({len(r['polish_warnings'])})" if is_id else f"03. Customer Screen & Human Copy ({len(r['polish_warnings'])})"
        lines.append(f"{bold}{hdr}{rst}")
        for i, w in enumerate(r["polish_warnings"], start=1):
            t = w["title_id"] if is_id else w["title_en"]
            loc = f" [{w['file']}:{w['line']}]" if w.get("file") else ""
            lines.append(f"  {amber}{i}. {t}{rst}{dim}{loc}{rst}")

    if r["blindspots"]:
        lines.append(thin)
        lines.append(f"{bold}{'04. Cek Mata Owner 2 Menit' if is_id else '04. Owner 2-Minute Manual Check'}{rst}")
        for bs in r["blindspots"]:
            lines.append(f"  - {bs['title_id'] if is_id else bs['title_en']}")

    lines.append(hr)
    short_seal = r["seal_sha256"][:16]
    if is_id:
        lines.append(f"Dokumen Lengkap : {html_path} (tambah --open bila ingin buka di browser)")
        lines.append(f"Segel SHA-256   : {short_seal}... | Git: {r['git_commit']}")
    else:
        lines.append(f"Full Dossier    : {html_path} (pass --open to view in browser)")
        lines.append(f"SHA-256 Seal    : {short_seal}... | Git: {r['git_commit']}")
    lines.append(hr)
    return "\n".join(lines) + "\n"


def render_markdown(r: dict[str, Any], html_path: Path, lang: str) -> str:
    is_id = lang.lower() == "id"
    uri = "file://" + html_path.resolve().as_posix()
    short_seal = r["seal_sha256"][:16]
    passed_probes = sum(1 for p in r["runtime_probes"] if p["executed"] and p["passed"])
    if is_id:
        return (
            f"### Hasil Pemeriksaan AxoProof: {r['headline_id']}\n\n"
            "| Pemeriksaan | Hasil | Rincian |\n"
            "| :--- | :--- | :--- |\n"
            f"| **Kasir, Privasi & Fitur Asli** | `{len(r['fatal_risks'])} temuan` | {r['files_scanned']} berkas diperiksa |\n"
            f"| **Uji Database & Pintu Live** | `{passed_probes} / {len(r['runtime_probes'])} lulus` | Segel SHA-256 `{short_seal}` |\n"
            f"| **Kerapian Layar & Bahasa** | `{len(r['polish_warnings'])} catatan` | [Buka Lembar Pemeriksaan Lengkap]({uri}) |\n"
        )
    return (
        f"### AxoProof Verification: {r['headline_en']}\n\n"
        "| Check | Result | Detail |\n"
        "| :--- | :--- | :--- |\n"
        f"| **Money, Privacy & Real Wiring** | `{len(r['fatal_risks'])} blocking` | {r['files_scanned']} source files inspected |\n"
        f"| **Live Server & DB Probes** | `{passed_probes} / {len(r['runtime_probes'])} proven` | SHA-256 seal `{short_seal}` |\n"
        f"| **Customer Screen & Copy** | `{len(r['polish_warnings'])} notes` | [Open Full Inspection Dossier]({uri}) |\n"
    )


def render_html(r: dict[str, Any], default_lang: str = "en") -> str:
    lang = "id" if default_lang.lower() == "id" else "en"
    tone = "tone-warn"
    if r["verdict"] == VERDICT_SAFE:
        tone = "tone-pass"
    elif r["verdict"] == VERDICT_HOLD:
        tone = "tone-fail"

    esc = html.escape
    has_repairs = bool(r["fatal_risks"] or r["polish_warnings"])

    if r["fatal_risks"]:
        fatal_items = []
        for idx, f in enumerate(r["fatal_risks"], start=1):
            loc_html = f'<span class="finding-loc">{esc(f["file"])}:{f["line"]}</span>' if f.get("file") else ""
            fatal_items.append(
                f'<li class="finding-row"><div>'
                f'<p class="finding-title"><span class="flag-fail">{idx}.</span>'
                f'<span class="lang-en">{esc(f["title_en"])}</span><span class="lang-id">{esc(f["title_id"])}</span></p>'
                f'<p class="finding-impact"><span class="lang-en">{esc(f["business_impact_en"])}</span>'
                f'<span class="lang-id">{esc(f["business_impact_id"])}</span></p></div>{loc_html}</li>'
            )
        fatal_block = '<ol class="finding-list">' + "".join(fatal_items) + "</ol>"
    else:
        fatal_block = (
            '<p class="clean-line">'
            '<span class="lang-en">Passed: no fake buttons, floating-point currency bugs, or unlocked endpoints detected.</span>'
            '<span class="lang-id">Bersih: tidak ditemukan tombol pajangan kosong, salah tipe uang, maupun pintu data terbuka.</span></p>'
        )

    if r["runtime_probes"]:
        probe_rows = []
        for p in r["runtime_probes"]:
            if p["executed"] and p["passed"]:
                st_html = '<span class="status-word-pass"><span class="lang-en">Proven</span><span class="lang-id">Terbukti</span></span>'
            elif p["executed"]:
                st_html = '<span class="status-word-fail"><span class="lang-en">Failed</span><span class="lang-id">Gagal</span></span>'
            else:
                st_html = '<span class="status-word-skip"><span class="lang-en">Pending</span><span class="lang-id">Tertunda</span></span>'
            probe_rows.append(
                f'<tr><td>{st_html}<div>{esc(p["name"])}</div></td>'
                f'<td><span class="lang-en">{esc(p["summary_en"])}</span><span class="lang-id">{esc(p["summary_id"])}</span></td>'
                f'<td class="probe-code">{esc(p["raw_evidence"])}</td></tr>'
            )
        probes_block = (
            '<table class="probe-table"><thead><tr>'
            '<th><span class="lang-en">Check</span><span class="lang-id">Pengujian</span></th>'
            '<th><span class="lang-en">Outcome</span><span class="lang-id">Hasil Nyata</span></th>'
            '<th><span class="lang-en">Measured Proof</span><span class="lang-id">Bukti Mesin</span></th>'
            '</tr></thead><tbody>' + "".join(probe_rows) + "</tbody></table>"
        )
    else:
        probes_block = (
            '<p class="finding-impact">'
            '<span class="lang-en">No live server URL was configured in .axoproof/claims.json yet. Static checks completed; live database write counts remain unmeasured.</span>'
            '<span class="lang-id">Alamat server lokal belum diisi di .axoproof/claims.json. Kode sudah diperiksa, namun pertambahan baris database belum diukur secara live.</span></p>'
        )

    if r["polish_warnings"]:
        pol_items = []
        for idx, w in enumerate(r["polish_warnings"], start=1):
            loc_html = f'<span class="finding-loc">{esc(w["file"])}:{w["line"]}</span>' if w.get("file") else ""
            pol_items.append(
                f'<li class="finding-row"><div>'
                f'<p class="finding-title"><span class="flag-warn">{idx}.</span>'
                f'<span class="lang-en">{esc(w["title_en"])}</span><span class="lang-id">{esc(w["title_id"])}</span></p>'
                f'<p class="finding-impact"><span class="lang-en">{esc(w["business_impact_en"])}</span>'
                f'<span class="lang-id">{esc(w["business_impact_id"])}</span></p></div>{loc_html}</li>'
            )
        polish_block = '<ol class="finding-list">' + "".join(pol_items) + "</ol>"
    else:
        polish_block = (
            '<p class="clean-line">'
            '<span class="lang-en">Clean: no fake sample contacts, dead \'#\' anchors, or robotic brochure copy detected.</span>'
            '<span class="lang-id">Rapi: tidak ada kontak contoh palsu, link \'#\' buntu, maupun kalimat promosi robotik.</span></p>'
        )

    bs_items = []
    for idx, b in enumerate(r["blindspots"], start=1):
        bs_items.append(
            f'<li class="finding-row"><div>'
            f'<p class="finding-title">{idx}. <span class="lang-en">{esc(b["title_en"])}</span><span class="lang-id">{esc(b["title_id"])}</span></p>'
            f'<p class="finding-impact"><span class="lang-en">{esc(b["why_en"])}</span><span class="lang-id">{esc(b["why_id"])}</span></p>'
            f"</div></li>"
        )
    blindspots_block = '<ol class="finding-list">' + "".join(bs_items) + "</ol>"

    if has_repairs:
        rep_lines = ["Fix every defect verified by AxoProof below, then re-run: python3 scripts/axoproof.py verify --sync-axoloom\n"]
        for f in r["fatal_risks"]:
            loc = f"{f['file']}:{f['line']} - " if f.get("file") else ""
            rep_lines.append(f"[BLOCKING / {f['id']}] {loc}{f['repair_instruction']}")
        for w in r["polish_warnings"]:
            loc = f"{w['file']}:{w['line']} - " if w.get("file") else ""
            rep_lines.append(f"[POLISH / {w['id']}] {loc}{w['repair_instruction']}")
        repair_text = esc("\n".join(rep_lines))
        copy_btn = (
            '<button type="button" class="copy-action" id="copy-repair-btn">'
            '<span class="lang-en">Copy AI repair brief</span>'
            '<span class="lang-id">Salin perintah untuk AI</span></button>'
        )
    else:
        repair_text = "No repair actions pending."
        copy_btn = ""

    axo = r["axoloom"]
    if axo.get("connected"):
        axo_dd = (
            f'<span class="lang-en">Connected ({axo["indexed_endpoints"]} routes, {axo["indexed_sql_tables"]} tables)</span>'
            f'<span class="lang-id">Terhubung ({axo["indexed_endpoints"]} route, {axo["indexed_sql_tables"]} tabel)</span>'
        )
    else:
        axo_dd = '<span class="lang-en">Standalone</span><span class="lang-id">Mandiri</span>'

    if tone == "tone-pass":
        status_dd = '<span class="lang-en">Verified live</span><span class="lang-id">Terbukti jalan</span>'
        status_cls = "val-pass"
    elif tone == "tone-fail":
        status_dd = '<span class="lang-en">Hold release</span><span class="lang-id">Tahan rilis</span>'
        status_cls = "val-fail"
    else:
        status_dd = '<span class="lang-en">Pending live check</span><span class="lang-id">Menunggu uji jalan</span>'
        status_cls = ""

    fatal_cls = "val-fail" if r["fatal_risks"] else "val-pass"
    en_pressed = "true" if lang == "en" else "false"
    id_pressed = "true" if lang == "id" else "false"

    return f"""<!DOCTYPE html>
<html lang="{lang}" data-lang="{lang}" data-surface="paper">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="Independent runtime proof, money safety, and human-readiness inspection ledger sealed by AxoProof.">
  <title>AxoProof Inspection Dossier: {esc(r['project_name'])}</title>
  <style>/* Hallmark - checked: 2026-11-28 - genre: editorial - shape: app screen
   * critique: P5 H5 E5 S5 R5 V5 - cut nested dashboard cards and dark neon boxes in favor of a warm archival inspection dossier with asymmetric two-column ledger rules and real tabular measurements
   * built: AxoProof - v0.2.0 */
    :root {{
      --paper: #f6f4ee;
      --paper-inset: #ece9e0;
      --ink: #191816;
      --ink-2: #49463f;
      --ink-3: #6b665c;
      --rule: #d8d3c8;
      --rule-strong: #191816;
      --brand: #5c45b9;
      --brand-press: #49359b;
      --on-brand: #f8f6ff;
      --pass-ink: #175937;
      --pass-wash: #e5efe9;
      --warn-ink: #7d490d;
      --warn-wash: #f2e9d8;
      --fail-ink: #991f26;
      --fail-wash: #f4e3e4;
      --code-bg: #ebe7dc;
      --night-paper: #151513;
      --night-inset: #1f1e1b;
      --night-ink: #f1eee7;
      --night-ink-2: #c5c0b4;
      --night-ink-3: #948e81;
      --night-rule: #2e2c27;
      --night-rule-strong: #e5e1d8;
      --night-brand: #8875e8;
      --night-brand-press: #725dd9;
      --night-pass-ink: #86d9ab;
      --night-pass-wash: #142a1f;
      --night-warn-ink: #ecc078;
      --night-warn-wash: #2b2111;
      --night-fail-ink: #f39ca1;
      --night-fail-wash: #2e1518;
      --night-code-bg: #1d1c19;
      --font-display: "Iowan Old Style", "Palatino Linotype", Charter, Georgia, serif;
      --font-body: "Avenir Next", "Segoe UI", "Helvetica Neue", sans-serif;
      --font-mono: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
      --space-1: 4px;
      --space-2: 8px;
      --space-3: 12px;
      --space-4: 16px;
      --space-5: 20px;
      --space-6: 24px;
      --space-8: 32px;
      --space-10: 40px;
      --space-16: 64px;
    }}
    html[data-surface="night"] {{
      --paper: var(--night-paper);
      --paper-inset: var(--night-inset);
      --ink: var(--night-ink);
      --ink-2: var(--night-ink-2);
      --ink-3: var(--night-ink-3);
      --rule: var(--night-rule);
      --rule-strong: var(--night-rule-strong);
      --brand: var(--night-brand);
      --brand-press: var(--night-brand-press);
      --on-brand: var(--night-paper);
      --pass-ink: var(--night-pass-ink);
      --pass-wash: var(--night-pass-wash);
      --warn-ink: var(--night-warn-ink);
      --warn-wash: var(--night-warn-wash);
      --fail-ink: var(--night-fail-ink);
      --fail-wash: var(--night-fail-wash);
      --code-bg: var(--night-code-bg);
    }}
    * {{ box-sizing: border-box; }}
    html, body {{
      margin: 0;
      padding: 0;
      width: 100%;
      overflow-x: clip;
      background: var(--paper);
      color: var(--ink);
      font-family: var(--font-body);
      font-size: 16px;
      line-height: 1.55;
      -webkit-font-smoothing: antialiased;
    }}
    .sheet {{
      max-width: 1040px;
      margin: 0 auto;
      padding: var(--space-8) var(--space-6) var(--space-16);
    }}
    .masthead {{
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: space-between;
      gap: var(--space-4);
      padding-bottom: var(--space-5);
      border-bottom: 2px solid var(--rule-strong);
    }}
    .identity {{
      display: flex;
      align-items: center;
      gap: var(--space-3);
      min-width: 0;
    }}
    .axo-emblem {{
      width: 32px;
      height: 32px;
      flex-shrink: 0;
      color: var(--brand);
    }}
    .doc-name {{
      margin: 0;
      font-size: 17px;
      font-weight: 700;
      color: var(--ink);
    }}
    .doc-sub {{
      margin: 0;
      font-size: 14px;
      color: var(--ink-3);
    }}
    .toolbar {{
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: var(--space-2);
    }}
    .seg-group {{
      display: inline-flex;
      border: 1px solid var(--rule);
      border-radius: 6px;
      padding: 2px;
      background: var(--paper-inset);
    }}
    .ctrl-btn {{
      min-height: 44px;
      min-width: 48px;
      padding: 0 var(--space-3);
      border: 1px solid transparent;
      border-radius: 4px;
      background: transparent;
      color: var(--ink-2);
      font-family: var(--font-body);
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
    }}
    .ctrl-btn:hover {{ color: var(--ink); }}
    .ctrl-btn:active {{ background: var(--rule); color: var(--ink); }}
    .ctrl-btn:disabled {{ opacity: 0.4; cursor: not-allowed; }}
    .ctrl-btn[aria-pressed="true"] {{
      background: var(--paper);
      color: var(--ink);
      border-color: var(--rule);
    }}
    .ctrl-btn:focus-visible, .copy-action:focus-visible, summary:focus-visible {{
      outline: 2px solid var(--brand);
      outline-offset: 2px;
    }}
    .verdict-spread {{
      display: grid;
      grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
      gap: var(--space-10);
      padding: var(--space-10) 0;
      border-bottom: 1px solid var(--rule-strong);
      align-items: start;
    }}
    .verdict-head {{
      margin: 0 0 var(--space-4) 0;
      font-family: var(--font-display);
      font-size: 36px;
      font-weight: 700;
      line-height: 1.12;
      color: var(--ink);
      text-wrap: balance;
      overflow-wrap: break-word;
      min-width: 0;
    }}
    .tone-pass .verdict-head {{ color: var(--pass-ink); }}
    .tone-warn .verdict-head {{ color: var(--warn-ink); }}
    .tone-fail .verdict-head {{ color: var(--fail-ink); }}
    .verdict-lede {{
      margin: 0;
      max-width: 58ch;
      font-size: 17px;
      color: var(--ink-2);
    }}
    .copy-action {{
      margin-top: var(--space-6);
      min-height: 44px;
      padding: 0 var(--space-5);
      border: 1px solid var(--brand);
      border-radius: 6px;
      background: var(--brand);
      color: var(--on-brand);
      font-family: var(--font-body);
      font-size: 15px;
      font-weight: 600;
      white-space: nowrap;
      cursor: pointer;
    }}
    .copy-action:hover {{ background: var(--brand-press); border-color: var(--brand-press); }}
    .copy-action:active {{ transform: translateY(1px); }}
    .copy-action:disabled {{ opacity: 0.45; cursor: not-allowed; }}
    .summary-ledger {{
      margin: 0;
      border-top: 1px solid var(--rule);
      font-variant-numeric: tabular-nums;
    }}
    .ledger-pair {{
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      gap: var(--space-4);
      padding: var(--space-3) 0;
      border-bottom: 1px solid var(--rule);
    }}
    .ledger-pair dt {{ font-size: 14px; color: var(--ink-2); }}
    .ledger-pair dd {{ margin: 0; font-size: 15px; font-weight: 700; color: var(--ink); text-align: right; }}
    .val-fail {{ color: var(--fail-ink); }}
    .val-pass {{ color: var(--pass-ink); }}
    .chapter {{
      display: grid;
      grid-template-columns: 240px minmax(0, 1fr);
      gap: var(--space-8);
      padding: var(--space-8) 0;
      border-bottom: 1px solid var(--rule);
    }}
    .chapter-head h2 {{
      margin: 0 0 var(--space-2) 0;
      font-family: var(--font-display);
      font-size: 21px;
      font-weight: 700;
      line-height: 1.25;
      color: var(--ink);
      text-wrap: balance;
      overflow-wrap: break-word;
      min-width: 0;
    }}
    .chapter-note {{ margin: 0; font-size: 14px; color: var(--ink-3); max-width: 32ch; }}
    .finding-list {{ list-style: none; margin: 0; padding: 0; border-top: 1px solid var(--rule); }}
    .finding-row {{
      padding: var(--space-4) 0;
      border-bottom: 1px solid var(--rule);
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: var(--space-4);
      align-items: baseline;
    }}
    .finding-title {{ margin: 0 0 var(--space-1) 0; font-size: 16px; font-weight: 700; color: var(--ink); }}
    .finding-title .flag-fail {{ color: var(--fail-ink); margin-right: var(--space-2); }}
    .finding-title .flag-warn {{ color: var(--warn-ink); margin-right: var(--space-2); }}
    .finding-impact {{ margin: 0; max-width: 64ch; font-size: 15px; color: var(--ink-2); }}
    .finding-loc {{ font-family: var(--font-mono); font-size: 13px; color: var(--ink-3); white-space: nowrap; }}
    .clean-line {{
      margin: 0;
      padding: var(--space-4) 0;
      border-top: 1px solid var(--rule);
      border-bottom: 1px solid var(--rule);
      font-size: 15px;
      font-weight: 600;
      color: var(--pass-ink);
    }}
    .probe-table-wrap {{ width: 100%; overflow-x: auto; }}
    table.probe-table {{
      width: 100%;
      border-collapse: collapse;
      border-top: 1px solid var(--rule);
      font-variant-numeric: tabular-nums;
    }}
    .probe-table th {{
      text-align: left;
      font-size: 13px;
      font-weight: 600;
      color: var(--ink-3);
      padding: var(--space-2) var(--space-3) var(--space-2) 0;
      border-bottom: 1px solid var(--rule);
    }}
    .probe-table td {{
      padding: var(--space-3) var(--space-3) var(--space-3) 0;
      border-bottom: 1px solid var(--rule);
      vertical-align: top;
      font-size: 15px;
    }}
    .probe-table td.probe-code {{ font-family: var(--font-mono); font-size: 13px; color: var(--ink-2); white-space: nowrap; }}
    .status-word-pass {{ color: var(--pass-ink); font-weight: 700; }}
    .status-word-fail {{ color: var(--fail-ink); font-weight: 700; }}
    .status-word-skip {{ color: var(--warn-ink); font-weight: 700; }}
    .agent-slip {{ margin-top: var(--space-8); padding-top: var(--space-4); border-top: 1px dashed var(--rule-strong); }}
    .agent-slip summary {{
      min-height: 44px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      cursor: pointer;
      font-size: 15px;
      font-weight: 700;
      color: var(--ink-2);
    }}
    pre.repair-brief {{
      margin: var(--space-3) 0 0 0;
      padding: var(--space-4);
      background: var(--code-bg);
      color: var(--ink);
      border: 1px solid var(--rule);
      border-radius: 4px;
      overflow-x: auto;
      font-family: var(--font-mono);
      font-size: 13px;
      line-height: 1.5;
    }}
    .notary-foot {{
      margin-top: var(--space-8);
      padding-top: var(--space-4);
      border-top: 1px solid var(--rule);
      display: flex;
      flex-wrap: wrap;
      justify-content: space-between;
      gap: var(--space-3);
      font-size: 13px;
      color: var(--ink-3);
    }}
    .notary-foot code {{ font-family: var(--font-mono); color: var(--ink-2); }}
    @media (max-width: 760px) {{
      .verdict-spread, .chapter {{ grid-template-columns: minmax(0, 1fr); gap: var(--space-5); }}
      .finding-row {{ grid-template-columns: minmax(0, 1fr); gap: var(--space-1); }}
      .verdict-head {{ font-size: 28px; }}
    }}
    html[data-lang="en"] .lang-id {{ display: none; }}
    html[data-lang="id"] .lang-en {{ display: none; }}
  </style>
</head>
<body class="{tone}">
  <main class="sheet">
    <header class="masthead">
      <div class="identity">
        <svg class="axo-emblem" viewBox="-36 -36 628 593" role="img" aria-label="AxoProof seal">
          <path fill="currentColor" fill-rule="evenodd" d="{AXOPROOF_ICON_PATH}"/>
        </svg>
        <div>
          <p class="doc-name">AxoProof <span class="lang-en">Inspection Record</span><span class="lang-id">Lembar Pemeriksaan Hasil Kerja AI</span></p>
          <p class="doc-sub">{esc(r['project_name'])} ({esc(r['generated_at_utc'])})</p>
        </div>
      </div>
      <div class="toolbar">
        <div class="seg-group" role="group" aria-label="Language">
          <button type="button" class="ctrl-btn" data-set-lang="en" aria-pressed="{en_pressed}">EN</button>
          <button type="button" class="ctrl-btn" data-set-lang="id" aria-pressed="{id_pressed}">ID</button>
        </div>
        <div class="seg-group" role="group" aria-label="Surface theme">
          <button type="button" class="ctrl-btn" data-set-surface="paper" aria-pressed="true"><span class="lang-en">Paper</span><span class="lang-id">Kertas</span></button>
          <button type="button" class="ctrl-btn" data-set-surface="night" aria-pressed="false"><span class="lang-en">Night</span><span class="lang-id">Malam</span></button>
        </div>
      </div>
    </header>

    <section class="verdict-spread" aria-live="polite">
      <div>
        <h1 class="verdict-head">
          <span class="lang-en">{esc(r['headline_en'])}</span>
          <span class="lang-id">{esc(r['headline_id'])}</span>
        </h1>
        <p class="verdict-lede">
          <span class="lang-en">{esc(r['subheadline_en'])}</span>
          <span class="lang-id">{esc(r['subheadline_id'])}</span>
        </p>
        {copy_btn}
      </div>
      <dl class="summary-ledger">
        <div class="ledger-pair">
          <dt><span class="lang-en">Release status</span><span class="lang-id">Status serah terima</span></dt>
          <dd class="{status_cls}">{status_dd}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Source files inspected</span><span class="lang-id">Berkas kode diperiksa</span></dt>
          <dd>{r['files_scanned']}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Blocking business or money defects</span><span class="lang-id">Masalah fatal bisnis &amp; uang</span></dt>
          <dd class="{fatal_cls}">{len(r['fatal_risks'])}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Customer screen &amp; language notes</span><span class="lang-id">Catatan tampilan &amp; bahasa pelanggan</span></dt>
          <dd>{len(r['polish_warnings'])}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Live HTTP &amp; database probes</span><span class="lang-id">Uji database &amp; pintu secara live</span></dt>
          <dd>{len(r['runtime_probes'])}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Axoloom memory link</span><span class="lang-id">Jembatan memori Axoloom</span></dt>
          <dd>{axo_dd}</dd>
        </div>
      </dl>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">01. Money, Privacy &amp; Real Wiring</span><span class="lang-id">01. Kasir, Kunci Data &amp; Fitur Asli</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Defects that lose orders, miscalculate currency, or leave private records open.</span>
          <span class="lang-id">Masalah yang bikin data pesanan hilang, salah hitung uang, atau data pelanggan bocor.</span>
        </p>
      </div>
      <div>{fatal_block}</div>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">02. Live Server &amp; Database Ledger</span><span class="lang-id">02. Bukti Kirim Data &amp; Isi Database</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Measured directly by AxoProof against your running app and database tables.</span>
          <span class="lang-id">Diukur langsung oleh AxoProof ke server dan tabel database nyata, bukan pengakuan AI.</span>
        </p>
      </div>
      <div class="probe-table-wrap">{probes_block}</div>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">03. Customer Screen &amp; Human Copy</span><span class="lang-id">03. Kerapian Layar &amp; Bahasa Manusia</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Checks for leftover sample contacts, dead links, robotic cliches, and cramped mobile buttons.</span>
          <span class="lang-id">Memeriksa sisa email contoh palsu, link buntu, bahasa brosur robotik, dan tombol HP kekecilan.</span>
        </p>
      </div>
      <div>{polish_block}</div>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">04. Owner 2-Minute Check</span><span class="lang-id">04. Cek Mata Owner 2 Menit</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Real-world items outside local automation that still need your own eyes before launch.</span>
          <span class="lang-id">Hal dunia nyata di luar jangkauan mesin lokal yang tetap wajib Anda pastikan sendiri.</span>
        </p>
      </div>
      {blindspots_block}
    </section>

    <details class="agent-slip">
      <summary>
        <span>
          <span class="lang-en">Technical repair slip for AI coding agent ({len(r['fatal_risks'])} blocking, {len(r['polish_warnings'])} polish)</span>
          <span class="lang-id">Slip instruksi teknis untuk AI Coding Agent ({len(r['fatal_risks'])} fatal, {len(r['polish_warnings'])} polesan)</span>
        </span>
      </summary>
      <pre class="repair-brief" id="repair-brief-text">{repair_text}</pre>
    </details>

    <footer class="notary-foot">
      <span>Git state: <code>{esc(r['git_commit'])}</code> ({r['git_dirty_files']} uncommitted)</span>
      <span>SHA-256 seal: <code>{esc(r['seal_sha256'])}</code></span>
    </footer>
  </main>

  <script>
    (function () {{
      var html = document.documentElement;
      var langBtns = document.querySelectorAll("[data-set-lang]");
      for (var i = 0; i < langBtns.length; i++) {{
        langBtns[i].addEventListener("click", function () {{
          var target = this.getAttribute("data-set-lang");
          html.setAttribute("data-lang", target);
          html.setAttribute("lang", target);
          for (var j = 0; j < langBtns.length; j++) {{
            langBtns[j].setAttribute("aria-pressed", langBtns[j].getAttribute("data-set-lang") === target ? "true" : "false");
          }}
        }});
      }}
      var surfBtns = document.querySelectorAll("[data-set-surface]");
      for (var k = 0; k < surfBtns.length; k++) {{
        surfBtns[k].addEventListener("click", function () {{
          var surf = this.getAttribute("data-set-surface");
          html.setAttribute("data-surface", surf);
          for (var m = 0; m < surfBtns.length; m++) {{
            surfBtns[m].setAttribute("aria-pressed", surfBtns[m].getAttribute("data-set-surface") === surf ? "true" : "false");
          }}
        }});
      }}
      var copyBtn = document.getElementById("copy-repair-btn");
      var briefEl = document.getElementById("repair-brief-text");
      if (copyBtn && briefEl && navigator.clipboard) {{
        copyBtn.addEventListener("click", function () {{
          var isID = html.getAttribute("data-lang") === "id";
          navigator.clipboard.writeText(briefEl.textContent || "").then(function () {{
            copyBtn.textContent = isID ? "Berhasil disalin! Tempel ke AI Agent" : "Copied! Paste into your AI agent";
          }});
        }});
      }}
    }})();
  </script>
</body>
</html>
"""


# ---------------------------------------------------------------------------
# Verdict Engine & CLI Entrypoints
# ---------------------------------------------------------------------------

def inspect_git_state(work_dir: Path) -> tuple[str, int]:
    sha = "uncommitted-workspace"
    dirty = 0
    try:
        proc = subprocess.run(["git", "-C", str(work_dir), "rev-parse", "--short", "HEAD"], capture_output=True, text=True, timeout=4, check=False)
        if proc.returncode == 0 and proc.stdout.strip():
            sha = proc.stdout.strip()
        st = subprocess.run(["git", "-C", str(work_dir), "status", "--porcelain"], capture_output=True, text=True, timeout=4, check=False)
        if st.returncode == 0:
            dirty = sum(1 for ln in st.stdout.splitlines() if ln.strip())
    except Exception:
        pass
    return sha, dirty


def compute_verdict(fatal: list[dict[str, Any]], polish: list[dict[str, Any]], probe_out: dict[str, Any]) -> tuple[str, str, str, str, str]:
    if fatal:
        return (
            VERDICT_HOLD,
            "Hold release until blocking defects are fixed",
            "Tahan rilis sebelum masalah utama dibereskan",
            f"AxoProof found {len(fatal)} issue(s) that could lose orders, miscalculate money, or expose private records. Click the button below to copy repair instructions for your AI agent.",
            f"AxoProof menemukan {len(fatal)} masalah yang bisa menghilangkan pesanan, melesetkan hitungan uang, atau membuka data pelanggan. Klik tombol di bawah untuk menyalin perintah perbaikan ke AI Agent.",
        )

    probes = probe_out["results"]
    has_persistence = any(p["kind"] == "db_persistence" and p["executed"] and p["passed"] for p in probes)
    all_passed = bool(probe_out["any_executed"] and probe_out["all_executed"] and all(p["executed"] and p["passed"] for p in probes))

    if all_passed and has_persistence:
        if polish:
            return (
                VERDICT_SAFE,
                "Verified live, with screen polish notes",
                "Terbukti jalan nyata, dengan catatan kerapian layar",
                f"Database writes, pricing rules, and login locks passed live checks. {len(polish)} customer-screen note(s) are listed in section 03.",
                f"Simpan database nyata, aturan harga, dan kunci login lulus uji langsung. Ada {len(polish)} catatan kerapian layar di bagian 03.",
            )
        return (
            VERDICT_SAFE,
            "Verified live and ready for customer testing",
            "Terbukti jalan nyata dan siap diuji ke pelanggan",
            "Every inspected code path, login lock, and live database row count passed independent measurement.",
            "Pemeriksaan kode, kunci pintu login, dan pertambahan isi database telah terbukti langsung oleh mesin.",
        )

    return (
        VERDICT_MANUAL,
        "Source code clean, live database check pending",
        "Kode bersih, menunggu uji simpan database langsung",
        "No static defects were found, while live server/database row counts have not been measured yet. Review section 04 before launch.",
        "Tidak ditemukan kode palsu atau celah statis, namun pertambahan baris database belum diukur secara live. Cek bagian 04 sebelum rilis.",
    )


def build_blindspots(claims: dict[str, Any], probe_out: dict[str, Any]) -> list[dict[str, str]]:
    items: list[dict[str, str]] = []
    for ext in claims.get("external_deps", []):
        svc = str(ext.get("service", "External service"))
        items.append({
            "title_en": svc,
            "title_id": svc,
            "why_en": str(ext.get("note_en", "")),
            "why_id": str(ext.get("note_id", ext.get("note_en", ""))),
        })
    if not probe_out["any_executed"]:
        items.append({
            "title_en": "Live click-to-database submission",
            "title_id": "Tes klik simpan data secara langsung",
            "why_en": "Start the app locally and wire .axoproof/claims.json BaseURL so AxoProof can verify real database row changes automatically.",
            "why_id": "Nyalakan server aplikasi dan isi BaseURL di .axoproof/claims.json agar AxoProof bisa menghitung pertambahan baris database otomatis.",
        })
    items.append({
        "title_en": "Real bank/Stripe settlement & business offer accuracy",
        "title_id": "Uang masuk rekening asli (Bank/Stripe/QRIS) & kesesuaian harga bisnis",
        "why_en": "Automated local probes never move real bank funds; always confirm one live payment and verify your real product pricing before public ads.",
        "why_id": "Pengujian lokal tidak memindahkan uang bank asli; selalu tes 1 pembayaran kecil nyata dan pastikan daftar harga sesuai katalog asli Anda.",
    })
    return items


def execute_verification(
    work_dir: Path,
    claims_path: str = "",
    scope: str = "all",
    lang: str = "en",
    sync_axoloom: bool = False,
    allow_remote: bool = False,
) -> tuple[dict[str, Any], Path, Path]:
    abs_dir = work_dir.resolve()
    claims = load_claims(abs_dir, claims_path)

    allowed: set[str] | None = None
    if scope.lower() == "diff":
        diff_set = resolve_git_diff_files(abs_dir)
        if diff_set:
            allowed = expand_blast_radius_files(abs_dir, diff_set)

    scan_res = scan_workspace(abs_dir, allowed)
    axo_ctx = inspect_axoloom_bridge(abs_dir, claims.get("protected_routes", []))
    probe_out = run_all_probes(abs_dir, claims, axo_ctx.get("undeclared_routes_found", []), allow_remote)

    all_fatal = list(scan_res["fatal_risks"]) + list(probe_out["fatal_findings"])
    blindspots = build_blindspots(claims, probe_out)
    verdict, h_en, h_id, s_en, s_id = compute_verdict(all_fatal, scan_res["polish_warnings"], probe_out)
    git_sha, git_dirty = inspect_git_state(abs_dir)

    rec: dict[str, Any] = {
        "schema_version": "1.0",
        "project_name": claims["project_name"],
        "workspace_summary": claims.get("summary", ""),
        "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "git_commit": git_sha,
        "git_dirty_files": git_dirty,
        "verdict": verdict,
        "headline_en": h_en,
        "headline_id": h_id,
        "subheadline_en": s_en,
        "subheadline_id": s_id,
        "files_scanned": scan_res["files_scanned"],
        "fatal_risks": all_fatal,
        "polish_warnings": scan_res["polish_warnings"],
        "runtime_probes": probe_out["results"],
        "blindspots": blindspots,
        "axoloom": axo_ctx,
        "seal_sha256": "",
    }

    if sync_axoloom:
        rec["seal_sha256"] = compute_seal(rec)
        rec["axoloom"]["writeback_path"] = sync_back_to_axoloom(abs_dir, rec)

    rec["seal_sha256"] = compute_seal(rec)

    out_dir = abs_dir / ".axoproof"
    out_dir.mkdir(parents=True, exist_ok=True)
    json_path = out_dir / "receipt.json"
    html_path = out_dir / "RECEIPT.html"

    json_path.write_text(json.dumps(rec, indent=2) + "\n", encoding="utf-8")
    html_path.write_text(render_html(rec, lang), encoding="utf-8")
    return rec, json_path, html_path


def _pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


def run_demo(open_browser: bool = False) -> int:
    """Runs a self-contained sabotage vs verified store showcase in a temporary workspace."""
    tmp = Path(tempfile.mkdtemp(prefix="axoproof-demo-"))
    sab_port = _pick_free_port()
    ver_port = _pick_free_port()
    while ver_port == sab_port:
        ver_port = _pick_free_port()

    sab_dir = tmp / "sabotaged-store"
    ver_dir = tmp / "verified-store"
    sab_dir.mkdir()
    ver_dir.mkdir()

    try:
        # 1. Sabotaged store (Python stdlib HTTP server + placebo bugs)
        (sab_dir / "dashboard.html").write_text(
            '<!DOCTYPE html><html><body>'
            '<h1>Unleash the Power of Seamlessly Integrated Retail</h1>'
            '<p>Trusted by 10,000+ stores! Email support@example.com</p>'
            '<div>Total Revenue: $18,500.00</div>'
            '<a href="#">Invoice</a>'
            '<button class="h-5" onClick={() => {}}>Sync</button>'
            '</body></html>\n',
            encoding="utf-8",
        )
        (sab_dir / "server.py").write_text(
            'import http.server, json, sqlite3\n'
            'conn = sqlite3.connect("store.db")\n'
            'conn.execute("CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY, sku TEXT, total REAL)")\n'
            'conn.close()\n'
            'http.server.HTTPServer.allow_reuse_address = True\n'
            'class H(http.server.BaseHTTPRequestHandler):\n'
            '    def log_message(self, *a): pass\n'
            '    def do_GET(self):\n'
            '        if self.path == "/":\n'
            '            self.send_response(200); self.end_headers(); self.wfile.write(open("dashboard.html","rb").read())\n'
            '        elif self.path == "/api/health":\n'
            '            self.send_response(200); self.end_headers(); self.wfile.write(b\'{"ok":true}\')\n'
            '        elif self.path == "/api/customers/orders":\n'
            '            self.send_response(200); self.end_headers(); self.wfile.write(b\'{"orders":[{"customer":"Alice"}]}\')\n'
            '    def do_POST(self):\n'
            '        length = int(self.headers.get("Content-Length", 0))\n'
            '        payload = json.loads(self.rfile.read(length) or b"{}")\n'
            '        price = float(payload["price"])\n'
            '        self.send_response(200); self.end_headers(); self.wfile.write(json.dumps({"paid": price}).encode())\n'
            f'http.server.HTTPServer(("127.0.0.1", {sab_port}), H).serve_forever()\n',
            encoding="utf-8",
        )
        (sab_dir / ".axoproof").mkdir()
        (sab_dir / ".axoproof" / "claims.json").write_text(
            json.dumps({
                "project_name": "Sabotaged Vibe-Coded Checkout Demo",
                "base_url": f"http://127.0.0.1:{sab_port}",
                "start_command": f'"{sys.executable}" server.py',
                "health_path": "/api/health",
                "screen_routes": ["/"],
                "persistence": [{
                    "name": "Submitting checkout order saves a real row in orders table",
                    "method": "POST", "path": "/api/orders",
                    "payload_json": '{"sku":"BEAN-1KG","quantity":2,"price":24.50}',
                    "sqlite_path": "./store.db", "table": "orders", "expect_min_delta": 1,
                }],
                "protected_routes": [{"name": "Customer order history blocks unauthenticated visitors", "method": "GET", "path": "/api/customers/orders"}],
                "money_tampers": [{
                    "name": "Checkout rejects negative quantity and $0.01 price tamper",
                    "method": "POST", "path": "/api/orders",
                    "tampered_body_json": '{"sku":"BEAN-1KG","quantity":-5,"price":0.01}',
                    "sqlite_path": "./store.db", "table": "orders",
                }],
            }, indent=2),
            encoding="utf-8",
        )

        # 2. Verified store (int cents, server catalog, durable SQLite insert, 401 auth, 400 tamper guard)
        (ver_dir / "dashboard.html").write_text(
            '<!DOCTYPE html><html><body>'
            '<h1>Fresh Roasted Single-Origin Beans</h1>'
            '<p>Orders placed before 15:00 UTC ship the same afternoon.</p>'
            '<a href="/orders/history">View My Orders</a>'
            '</body></html>\n',
            encoding="utf-8",
        )
        (ver_dir / "server.py").write_text(
            'import http.server, json, sqlite3\n'
            'CATALOG = {"BEAN-1KG": 2450}\n'
            'conn = sqlite3.connect("verified.db")\n'
            'conn.execute("CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY, sku TEXT, qty INTEGER, total_cents INTEGER)")\n'
            'conn.commit(); conn.close()\n'
            'http.server.HTTPServer.allow_reuse_address = True\n'
            'class H(http.server.BaseHTTPRequestHandler):\n'
            '    def log_message(self, *a): pass\n'
            '    def do_GET(self):\n'
            '        if self.path == "/":\n'
            '            self.send_response(200); self.end_headers(); self.wfile.write(open("dashboard.html","rb").read())\n'
            '        elif self.path == "/api/health":\n'
            '            self.send_response(200); self.end_headers(); self.wfile.write(b\'{"ok":true}\')\n'
            '        elif self.path == "/api/customers/orders":\n'
            '            if not self.headers.get("Authorization","").startswith("Bearer "):\n'
            '                self.send_response(401); self.end_headers(); return\n'
            '            self.send_response(200); self.end_headers(); self.wfile.write(b\'{"orders":[]}\')\n'
            '    def do_POST(self):\n'
            '        if self.path != "/api/orders": self.send_response(404); self.end_headers(); return\n'
            '        data = json.loads(self.rfile.read(int(self.headers.get("Content-Length",0))) or b"{}")\n'
            '        qty = int(data.get("quantity", 0))\n'
            '        sku = data.get("sku", "")\n'
            '        if qty <= 0 or sku not in CATALOG:\n'
            '            self.send_response(400); self.end_headers(); return\n'
            '        c = sqlite3.connect("verified.db")\n'
            '        c.execute("INSERT INTO orders (sku, qty, total_cents) VALUES (?, ?, ?)", (sku, qty, qty * CATALOG[sku]))\n'
            '        c.commit(); c.close()\n'
            '        self.send_response(201); self.end_headers(); self.wfile.write(b\'{"saved":true}\')\n'
            f'http.server.HTTPServer(("127.0.0.1", {ver_port}), H).serve_forever()\n',
            encoding="utf-8",
        )
        (ver_dir / ".axoproof").mkdir()
        (ver_dir / ".axoproof" / "claims.json").write_text(
            json.dumps({
                "project_name": "Verified Coffee Roastery Checkout",
                "base_url": f"http://127.0.0.1:{ver_port}",
                "start_command": f'"{sys.executable}" server.py',
                "health_path": "/api/health",
                "screen_routes": ["/"],
                "persistence": [{
                    "name": "Submitting checkout order saves a real row in orders table",
                    "method": "POST", "path": "/api/orders",
                    "payload_json": '{"sku":"BEAN-1KG","quantity":2}',
                    "sqlite_path": "./verified.db", "table": "orders", "expect_min_delta": 1,
                }],
                "protected_routes": [{"name": "Customer order history blocks unauthenticated visitors", "method": "GET", "path": "/api/customers/orders"}],
                "money_tampers": [{
                    "name": "Checkout rejects negative quantity exploit",
                    "method": "POST", "path": "/api/orders",
                    "tampered_body_json": '{"sku":"BEAN-1KG","quantity":-5}',
                    "sqlite_path": "./verified.db", "table": "orders",
                }],
            }, indent=2),
            encoding="utf-8",
        )

        sab_rec, _, sab_html = execute_verification(sab_dir)
        ver_rec, _, ver_html = execute_verification(ver_dir)
        sys.stdout.write(render_terminal(sab_rec, sab_html, "en", sys.stdout.isatty()))
        sys.stdout.write("\n")
        sys.stdout.write(render_terminal(ver_rec, ver_html, "en", sys.stdout.isatty()))
        if open_browser:
            webbrowser.open(ver_html.resolve().as_uri())
        return 0
    finally:
        if not open_browser:
            shutil.rmtree(tmp, ignore_errors=True)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="axoproof", description="AxoProof independent verification & Owner Receipt engine")
    sub = parser.add_subparsers(dest="cmd")

    sub.add_parser("version", help="Print AxoProof version")

    p_init = sub.add_parser("init", help="Create starter .axoproof/claims.json")
    p_init.add_argument("--dir", default=".", help="Workspace directory")

    p_ver = sub.add_parser("verify", help="Verify workspace and emit sealed RECEIPT.html")
    p_ver.add_argument("--dir", default=".", help="Workspace directory")
    p_ver.add_argument("--claims", default="", help="Optional path to claims.json")
    p_ver.add_argument("--scope", choices=["all", "diff"], default="all", help="Inspect all files or git diff + Axoloom callers")
    p_ver.add_argument("--lang", choices=["en", "id"], default="en", help="Default receipt language")
    p_ver.add_argument("--format", choices=["table", "markdown", "json"], default="table", help="Stdout output format")
    p_ver.add_argument("--json", action="store_true", help="Shorthand for --format json")
    p_ver.add_argument("--sync-axoloom", action="store_true", help="Write verification summary to Axoloom handoff memory")
    p_ver.add_argument("--open", action="store_true", help="Open RECEIPT.html in desktop browser")
    p_ver.add_argument("--allow-remote-probe", action="store_true", help="Allow non-loopback staging URLs")

    p_seal = sub.add_parser("check-seal", help="Verify cryptographic SHA-256 seal on receipt.json")
    p_seal.add_argument("--dir", default=".", help="Workspace directory")
    p_seal.add_argument("receipt_file", nargs="?", default="", help="Optional path to receipt.json")

    p_demo = sub.add_parser("demo", help="Run built-in sabotaged vs verified store showcase")
    p_demo.add_argument("--open", action="store_true", help="Open verified dossier in browser")

    args = parser.parse_args(argv)
    if args.cmd == "version":
        print(f"axoproof {VERSION}")
        return 0
    if args.cmd == "init":
        out = write_claims_template(Path(args.dir))
        print(f"Initialized starter claims manifest at {out}")
        return 0
    if args.cmd == "demo":
        return run_demo(open_browser=args.open)
    if args.cmd == "check-seal":
        target = Path(args.receipt_file) if args.receipt_file else Path(args.dir) / ".axoproof" / "receipt.json"
        data = json.loads(target.read_text(encoding="utf-8"))
        if not verify_seal(data):
            print(f"TAMPERED OR INVALID SEAL on {target}", file=sys.stderr)
            return 2
        print(f"VALID SHA-256 SEAL ({data['seal_sha256'][:16]}) for project {data['project_name']!r} [{data['verdict']}]")
        return 0
    if args.cmd == "verify":
        rec, _, html_path = execute_verification(
            work_dir=Path(args.dir),
            claims_path=args.claims,
            scope=args.scope,
            lang=args.lang,
            sync_axoloom=args.sync_axoloom,
            allow_remote=args.allow_remote_probe,
        )
        if args.open:
            webbrowser.open(html_path.resolve().as_uri())
        fmt = "json" if args.json else args.format
        if fmt == "json":
            print(json.dumps(rec, indent=2))
        elif fmt == "markdown":
            sys.stdout.write(render_markdown(rec, html_path, args.lang))
        else:
            sys.stdout.write(render_terminal(rec, html_path, args.lang, sys.stdout.isatty()))
        return 2 if rec["verdict"] == VERDICT_HOLD else 0

    parser.print_help()
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
