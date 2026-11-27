package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Intentionally sabotaged demo checkout server showcasing the 5 classic AI "vibe-coding" traps
// caught automatically by AxoProof:
// 1. Float64 currency rounding + trusting client-submitted price directly
// 2. Ghost checkout handler (returns 200 OK "paid" WITHOUT inserting a row into SQLite!)
// 3. Unlocked customer data route (/api/customers/orders returns 200 OK with PII without login!)
// 4. Negative quantity exploit accepted on /api/orders
// 5. Dashboard HTML with hardcoded fake revenue ("Rp 18.500.000"), example.com, href="#", & brochure slop

type OrderInput struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"` // TRAP: float64 money + client-submitted price
}

func main() {
	addr := ":18089"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}
	dbPath := filepath.Join(".", "store.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sku TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		total REAL NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// TRAP 2 & 4: Returns 200 OK pretending order saved, never inserts into SQLite, and accepts negative quantities!
	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		var in OrderInput
		_ = json.NewDecoder(r.Body).Decode(&in)
		total := float64(in.Quantity) * in.Price
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"paid","charged_total":%.2f}`, total)
	})

	// TRAP 3: Sensitive customer endpoint with zero authentication check!
	mux.HandleFunc("/api/customers/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orders":[{"customer":"Budi","phone":"0811992233","item":"VIP Package"}]}`))
	})

	log.Printf("Sabotaged demo store listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
