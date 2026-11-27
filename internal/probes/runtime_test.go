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

func TestReadBackPersistenceWorksWithoutSQLiteForPostgresOrSupabaseAPIs(t *testing.T) {
	ws := t.TempDir()

	// 1. Placebo server: POST returns 201, yet GET /api/orders stays empty!
	placeboAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"orders":[]}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer placeboAPI.Close()

	m := claims.Manifest{
		ProjectName: "postgres-api",
		BaseURL:     placeboAPI.URL,
		HealthPath:  "/api/health",
		Persistence: []claims.PersistenceClaim{
			{
				Name:                "Order persists via read-back",
				Method:              "POST",
				Path:                "/api/orders",
				PayloadJSON:         `{"sku":"PG-SKU-77","qty":1}`,
				ReadBackPath:        "/api/orders",
				ExpectReadSubstring: "PG-SKU-77",
				ExpectMinDelta:      1,
			},
		},
	}

	bad := probes.RunAll(ws, m, nil, false)
	if len(bad.FatalFindings) != 1 {
		t.Fatalf("expected placebo API without real persistence to fail read-back, got %+v", bad.FatalFindings)
	}

	// 2. Real persisting API: POST appends item so subsequent GET returns PG-SKU-77
	var items []string
	realAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			enc, _ := json.Marshal(map[string]any{"orders": items})
			_, _ = w.Write(enc)
			return
		}
		items = append(items, "PG-SKU-77")
		w.WriteHeader(http.StatusCreated)
	}))
	defer realAPI.Close()

	m.BaseURL = realAPI.URL
	good := probes.RunAll(ws, m, nil, false)
	if len(good.FatalFindings) != 0 {
		t.Fatalf("expected persisting API to pass read-back proof, got %+v", good.FatalFindings)
	}
}
