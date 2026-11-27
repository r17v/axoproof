package axobridge_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/r17v/axoproof/internal/axobridge"
	"github.com/r17v/axoproof/internal/claims"
	_ "modernc.org/sqlite"
)

func TestAxoloomBridgeCatchesUndeclaredSensitiveEndpoint(t *testing.T) {
	ws := t.TempDir()
	axoDir := filepath.Join(ws, ".axoloom")
	if err := os.MkdirAll(axoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(axoDir, "axoloom.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE nodes (id TEXT PRIMARY KEY, kind TEXT, name TEXT, project TEXT, path TEXT);
		CREATE TABLE guardrails (id INTEGER PRIMARY KEY, project TEXT, source_path TEXT, title TEXT, body TEXT);
		CREATE TABLE directives (id INTEGER PRIMARY KEY, scope TEXT, category TEXT, title TEXT, body TEXT);
		INSERT INTO nodes VALUES ('e1', 'endpoint', 'GET /api/customers/secret-export', 'demo', '/tmp/demo/api.go');
		INSERT INTO nodes VALUES ('t1', 'table', 'orders', 'demo', '/tmp/demo/schema.sql');
		INSERT INTO guardrails VALUES (1, 'demo', 'Spec.md', 'Integer Money', 'Use int64 cents.');
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	ctx := axobridge.Inspect(ws, []claims.ProtectedRouteClaim{})
	if !ctx.Connected {
		t.Fatalf("expected Axoloom bridge to connect to .axoloom/axoloom.db")
	}
	if ctx.IndexedSQLTables != 1 || ctx.IndexedEndpoints != 1 {
		t.Fatalf("unexpected counts: %+v", ctx)
	}
	if len(ctx.UndeclaredRoutesFound) != 1 || ctx.UndeclaredRoutesFound[0] != "/api/customers/secret-export" {
		t.Fatalf("expected bridge to surface undeclared sensitive route, got %+v", ctx.UndeclaredRoutesFound)
	}
}
