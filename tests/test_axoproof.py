#!/usr/bin/env python3
"""Unit and live counterfactual integration tests for AxoProof (100% Python stdlib)."""

from __future__ import annotations

import http.server
import json
import sqlite3
import subprocess
import sys
import tempfile
import threading
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

import axoproof  # noqa: E402


class TestStaticScannerAndDiffScope(unittest.TestCase):
    def test_separates_fatal_risks_from_polish_and_deduplicates(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "checkout.tsx").write_text(
                'export function Checkout() {\n'
                '  const total = parseFloat(req.body.price);\n'
                '  const tax = parseFloat(req.body.amount);\n'
                '  return (\n'
                '    <div>\n'
                '      <p>Trusted by 10,000+ teams! Contact support@example.com</p>\n'
                '      <div>Total Revenue: $18,500.00</div>\n'
                '      <a href="#">Terms</a>\n'
                '      <button className="h-5" onClick={() => {}}>Pay</button>\n'
                '    </div>\n'
                '  );\n'
                '}\n',
                encoding="utf-8",
            )
            (root / "api.py").write_text(
                'def handle(email, payload):\n'
                '    unit = payload["price"]\n'
                '    q = f"SELECT * FROM users WHERE email = {email}"\n'
                '    return q, unit\n',
                encoding="utf-8",
            )
            (root / "checkout.spec.ts").write_text(
                'it.skip("saves order row", () => {});\n',
                encoding="utf-8",
            )

            res = axoproof.scan_workspace(root)
            self.assertEqual(res["files_scanned"], 3)
            fatal_ids = {f["id"] for f in res["fatal_risks"]}
            polish_ids = {w["id"] for w in res["polish_warnings"]}

            self.assertIn("MONEY_FLOAT_ROUNDING", fatal_ids)
            self.assertIn("MONEY_CLIENT_PRICE_TRUST", fatal_ids)
            self.assertIn("DOOR_RAW_SQL_INJECTION", fatal_ids)
            self.assertIn("PLACEBO_DEAD_BUTTON", fatal_ids)
            self.assertIn("PLACEBO_HARDCODED_FIGURES", fatal_ids)
            self.assertIn("PLACEBO_SKIPPED_TEST", fatal_ids)

            self.assertIn("POLISH_FAKE_PLACEHOLDER_TRUST", polish_ids)
            self.assertIn("POLISH_DEAD_LINK", polish_ids)
            self.assertIn("POLISH_TINY_MOBILE_BUTTON", polish_ids)

            # Deduplication: two float money lines in checkout.tsx collapse into 1 finding with count note
            float_hits = [f for f in res["fatal_risks"] if f["id"] == "MONEY_FLOAT_ROUNDING" and f["file"] == "checkout.tsx"]
            self.assertEqual(len(float_hits), 1)
            self.assertIn("occurs 2 times", float_hits[0]["repair_instruction"])

    def test_git_diff_scope_scans_only_modified_files(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            subprocess.run(["git", "init", "-q"], cwd=tmp, check=True)
            subprocess.run(["git", "config", "user.email", "test@invalid.local"], cwd=tmp, check=True)
            subprocess.run(["git", "config", "user.name", "Tester"], cwd=tmp, check=True)

            (root / "legacy.py").write_text('legacy_price = float("19.99")\n', encoding="utf-8")
            (root / "clean.py").write_text("PRICE_CENTS = 1999\n", encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=tmp, check=True)
            subprocess.run(["git", "commit", "-q", "-m", "initial"], cwd=tmp, check=True)

            # Modify only clean.py
            (root / "clean.py").write_text("PRICE_CENTS = 2499\n", encoding="utf-8")
            diff_files = axoproof.resolve_git_diff_files(root)
            self.assertEqual(diff_files, {"clean.py"})

            scoped = axoproof.scan_workspace(root, allowed_rel=diff_files)
            self.assertEqual(scoped["files_scanned"], 1)
            self.assertEqual(scoped["fatal_risks"], [])


class TestCounterfactualRuntimeProbes(unittest.TestCase):
    def test_catches_ghost_write_unlocked_door_and_negative_tamper(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            db_path = root / "sabotaged.db"
            conn = sqlite3.connect(str(db_path))
            conn.execute("CREATE TABLE orders (id INTEGER PRIMARY KEY, sku TEXT)")
            conn.commit()
            conn.close()

            class SabotagedHandler(http.server.BaseHTTPRequestHandler):
                def log_message(self, *args):
                    pass

                def do_GET(self):
                    if self.path == "/api/health":
                        self.send_response(200)
                        self.end_headers()
                        self.wfile.write(b'{"ok":true}')
                    elif self.path == "/api/customers/orders":
                        # Bug: returns 200 without login!
                        self.send_response(200)
                        self.end_headers()
                        self.wfile.write(b'{"orders":["secret"]}')

                def do_POST(self):
                    # Bug: returns 200 on everything (including negative qty) and writes 0 rows!
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b'{"saved":true}')

            srv = http.server.HTTPServer(("127.0.0.1", 0), SabotagedHandler)
            port = srv.server_address[1]
            t = threading.Thread(target=srv.serve_forever, daemon=True)
            t.start()
            try:
                claims = {
                    "base_url": f"http://127.0.0.1:{port}",
                    "health_path": "/api/health",
                    "persistence": [{
                        "name": "Save order",
                        "method": "POST",
                        "path": "/api/orders",
                        "payload_json": '{"sku":"COFFEE","quantity":1}',
                        "sqlite_path": "sabotaged.db",
                        "table": "orders",
                    }],
                    "protected_routes": [{
                        "name": "Customer orders locked",
                        "method": "GET",
                        "path": "/api/customers/orders",
                    }],
                    "money_tampers": [{
                        "name": "Reject negative qty",
                        "method": "POST",
                        "path": "/api/orders",
                        "tampered_body_json": '{"sku":"COFFEE","quantity":-4}',
                        "sqlite_path": "sabotaged.db",
                        "table": "orders",
                    }],
                }
                out = axoproof.run_all_probes(root, claims, [], allow_remote=False)
                finding_ids = {f["id"] for f in out["fatal_findings"]}
                self.assertEqual(
                    finding_ids,
                    {"RUNTIME_GHOST_PERSISTENCE", "RUNTIME_UNLOCKED_DOOR", "RUNTIME_MONEY_TAMPER_ACCEPTED"},
                )
            finally:
                srv.shutdown()
                srv.server_close()

    def test_postgres_style_http_readback_persistence_without_sqlite(self) -> None:
        memory_orders: list[dict[str, object]] = []

        class ReadBackHandler(http.server.BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_GET(self):
                if self.path == "/api/health":
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b'{"ok":true}')
                elif self.path == "/api/orders":
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(json.dumps({"items": memory_orders}).encode("utf-8"))

            def do_POST(self):
                length = int(self.headers.get("Content-Length", "0"))
                body = json.loads(self.rfile.read(length) or b"{}")
                if int(body.get("quantity", 0)) <= 0:
                    self.send_response(422)
                    self.end_headers()
                    return
                memory_orders.append({"sku": body["sku"], "qty": body["quantity"]})
                self.send_response(201)
                self.end_headers()
                self.wfile.write(b'{"ok":true}')

        with tempfile.TemporaryDirectory() as tmp:
            srv = http.server.HTTPServer(("127.0.0.1", 0), ReadBackHandler)
            port = srv.server_address[1]
            t = threading.Thread(target=srv.serve_forever, daemon=True)
            t.start()
            try:
                claims = {
                    "base_url": f"http://127.0.0.1:{port}",
                    "health_path": "/api/health",
                    "persistence": [{
                        "name": "Postgres read-back persistence",
                        "method": "POST",
                        "path": "/api/orders",
                        "payload_json": '{"sku":"ROAST-PG-99","quantity":2}',
                        "read_back_path": "/api/orders",
                        "expect_read_substring": "ROAST-PG-99",
                        "expect_min_delta": 1,
                    }],
                    "money_tampers": [{
                        "name": "Negative quantity rejected",
                        "method": "POST",
                        "path": "/api/orders",
                        "tampered_body_json": '{"sku":"ROAST-PG-99","quantity":-2}',
                    }],
                }
                out = axoproof.run_all_probes(Path(tmp), claims, [], allow_remote=False)
                self.assertEqual(out["fatal_findings"], [])
                self.assertTrue(all(r["passed"] for r in out["results"]))
            finally:
                srv.shutdown()
                srv.server_close()


class TestAxoloomBridgeAndCryptographicSeal(unittest.TestCase):
    def test_axoloom_index_db_discovers_undeclared_routes_and_blast_radius(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve()
            axo_dir = root / ".axoloom"
            axo_dir.mkdir()
            db_path = axo_dir / "index.db"

            conn = sqlite3.connect(str(db_path))
            conn.executescript(
                """
                CREATE TABLE nodes (
                    id TEXT PRIMARY KEY, kind TEXT, name TEXT, title TEXT,
                    project TEXT, path TEXT
                );
                CREATE TABLE edges (
                    src_id TEXT, dst_name TEXT, dst_id TEXT, rel TEXT, path TEXT, line INTEGER
                );
                CREATE TABLE guardrails (
                    id INTEGER PRIMARY KEY, project TEXT, source_path TEXT, title TEXT, body TEXT
                );
                CREATE TABLE directives (
                    id INTEGER PRIMARY KEY, scope TEXT, category TEXT, title TEXT, body TEXT
                );
                """
            )
            conn.execute(
                "INSERT INTO nodes VALUES ('n1', 'endpoint', 'GET /api/patients/records', 'Records', ?, ?)",
                (root.name, str(root / "routes.py")),
            )
            conn.execute(
                "INSERT INTO nodes VALUES ('n2', 'table', 'orders', 'Orders Table', ?, ?)",
                (root.name, str(root / "schema.sql")),
            )
            conn.execute(
                "INSERT INTO nodes VALUES ('n3', 'function', 'calc_price', 'calc_price', ?, ?)",
                (root.name, str(root / "pricing.py")),
            )
            conn.execute(
                "INSERT INTO edges VALUES ('caller1', 'calc_price', 'n3', 'calls', ?, 14)",
                (str(root / "checkout_handler.py"),),
            )
            conn.execute(
                "INSERT INTO guardrails VALUES (1, ?, ?, 'Integer cents', 'Always use cents')",
                (root.name, str(root / "Index.md")),
            )
            conn.execute("INSERT INTO directives VALUES (1, 'global', 'security', 'Lock routes', 'Auth first')")
            conn.commit()
            conn.close()

            ctx = axoproof.inspect_axoloom_bridge(root, declared_protected=[])
            self.assertTrue(ctx["connected"])
            self.assertEqual(ctx["db_path"], ".axoloom/index.db")
            self.assertEqual(ctx["indexed_sql_tables"], 1)
            self.assertEqual(ctx["active_guardrails"], 1)
            self.assertEqual(ctx["active_directives"], 1)
            self.assertEqual(ctx["undeclared_routes_found"], ["/api/patients/records"])

            expanded = axoproof.expand_blast_radius_files(root, {"pricing.py"})
            self.assertEqual(expanded, {"pricing.py", "checkout_handler.py"})

    def test_sha256_seal_detects_tampering_and_renders_all_three_surfaces(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "app.py").write_text("UNIT_PRICE_CENTS = 1500\n", encoding="utf-8")
            rec, json_path, html_path = axoproof.execute_verification(root)

            self.assertEqual(rec["verdict"], axoproof.VERDICT_MANUAL)
            self.assertTrue(axoproof.verify_seal(rec))
            self.assertTrue(json_path.is_file())
            self.assertTrue(html_path.is_file())

            # Tamper with verdict -> seal verification MUST reject
            forged = json.loads(json_path.read_text(encoding="utf-8"))
            forged["verdict"] = axoproof.VERDICT_SAFE
            self.assertFalse(axoproof.verify_seal(forged))

            term_out = axoproof.render_terminal(rec, html_path, "en", color=False)
            md_out = axoproof.render_markdown(rec, html_path, "en")
            html_out = html_path.read_text(encoding="utf-8")

            self.assertIn("AxoProof Inspection Ledger", term_out)
            self.assertIn("### AxoProof Verification:", md_out)
            self.assertIn("/* Hallmark - checked:", html_out)
            self.assertIn('data-set-lang="en"', html_out)
            self.assertIn('data-set-surface="night"', html_out)


if __name__ == "__main__":
    unittest.main()
