package probes_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/r17v/axoproof/internal/claims"
	"github.com/r17v/axoproof/internal/probes"
	_ "modernc.org/sqlite"
)

func TestCounterfactualProbesCatchGhostWriteAndUnlockedDoors(t *testing.T) {
	ws := t.TempDir()
	dbFile := filepath.Join(ws, "test.db")
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE orders (id INTEGER PRIMARY KEY AUTOINCREMENT, sku TEXT, qty INTEGER)`); err != nil {
		t.Fatal(err)
	}

	// 1. Lying server: returns 200 OK without writing to DB, opens /api/customers, accepts negative qty
	lyingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer lyingSrv.Close()

	m := claims.Manifest{
		ProjectName: "test-app",
		BaseURL:     lyingSrv.URL,
		HealthPath:  "/api/health",
		Persistence: []claims.PersistenceClaim{
			{
				Name:           "Save order",
				Method:         "POST",
				Path:           "/api/orders",
				PayloadJSON:    `{"sku":"A","qty":1}`,
				SQLitePath:     "test.db",
				Table:          "orders",
				ExpectMinDelta: 1,
			},
		},
		ProtectedRoutes: []claims.ProtectedRouteClaim{
			{Name: "Customer list", Method: "GET", Path: "/api/customers"},
		},
		MoneyTampers: []claims.MoneyTamperClaim{
			{
				Name:             "Reject negative qty",
				Method:           "POST",
				Path:             "/api/orders",
				TamperedBodyJSON: `{"sku":"A","qty":-5}`,
				SQLitePath:       "test.db",
				Table:            "orders",
			},
		},
	}

	out := probes.RunAll(ws, m, nil, false)
	if len(out.FatalFindings) != 3 {
		t.Fatalf("expected 3 runtime fatal findings (ghost write, unlocked door, tamper accepted), got %d: %+v", len(out.FatalFindings), out.FatalFindings)
	}

	// 2. Honest server: inserts real row, rejects unauthenticated 401, rejects negative qty 400
	honestSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
		case "/api/customers":
			w.WriteHeader(http.StatusUnauthorized)
		case "/api/orders":
			var body struct {
				SKU string `json:"sku"`
				Qty int    `json:"qty"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Qty <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = db.Exec(`INSERT INTO orders (sku, qty) VALUES (?, ?)`, body.SKU, body.Qty)
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer honestSrv.Close()

	m.BaseURL = honestSrv.URL
	goodOut := probes.RunAll(ws, m, nil, false)
	if len(goodOut.FatalFindings) != 0 {
		t.Fatalf("expected 0 fatal findings on honest server, got %+v", goodOut.FatalFindings)
	}
	if !goodOut.AllExecuted {
		t.Fatalf("expected all probes executed on honest server")
	}
}
