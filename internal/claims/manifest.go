package claims

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PersistenceClaim declares an end-to-end write action that must change real database rows.
type PersistenceClaim struct {
	Name            string            `json:"name"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Headers         map[string]string `json:"headers,omitempty"`
	PayloadJSON     string            `json:"payload_json"`
	SQLitePath      string            `json:"sqlite_path"`
	Table           string            `json:"table"`
	ExpectMinDelta  int               `json:"expect_min_delta"`
	ExpectStatusMax int               `json:"expect_status_max,omitempty"`
}

// ProtectedRouteClaim declares a private endpoint that must reject unauthenticated strangers (401/403).
type ProtectedRouteClaim struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

// MoneyTamperClaim declares a transaction endpoint that must reject negative quantities or tampered prices.
type MoneyTamperClaim struct {
	Name              string            `json:"name"`
	Method            string            `json:"method"`
	Path              string            `json:"path"`
	Headers           map[string]string `json:"headers,omitempty"`
	TamperedBodyJSON  string            `json:"tampered_body_json"`
	SQLitePath        string            `json:"sqlite_path,omitempty"`
	Table             string            `json:"table,omitempty"`
}

// ExternalBlindspot lets the agent declare third-party integrations that require a 2-minute human check.
type ExternalBlindspot struct {
	Service string `json:"service"`
	NoteEN  string `json:"note_en"`
	NoteID  string `json:"note_id"`
}

// Manifest represents .axoproof/claims.json authored by the coding agent before verification.
type Manifest struct {
	ProjectName     string                `json:"project_name"`
	Summary         string                `json:"summary"`
	BaseURL         string                `json:"base_url,omitempty"`
	StartCommand    string                `json:"start_command,omitempty"`
	HealthPath      string                `json:"health_path,omitempty"`
	TouchesMoney    bool                  `json:"touches_money"`
	TouchesAuth     bool                  `json:"touches_auth"`
	Persistence     []PersistenceClaim    `json:"persistence,omitempty"`
	ProtectedRoutes []ProtectedRouteClaim `json:"protected_routes,omitempty"`
	MoneyTampers    []MoneyTamperClaim    `json:"money_tampers,omitempty"`
	ExternalDeps    []ExternalBlindspot   `json:"external_deps,omitempty"`
}

// DefaultPath returns the canonical claims manifest path inside a workspace.
func DefaultPath(workDir string) string {
	return filepath.Join(workDir, ".axoproof", "claims.json")
}

// Load reads .axoproof/claims.json if present, or synthesizes a clean zero-config default manifest.
func Load(workDir string, explicitPath string) (Manifest, bool, error) {
	target := explicitPath
	if strings.TrimSpace(target) == "" {
		target = DefaultPath(workDir)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			abs, _ := filepath.Abs(workDir)
			return Manifest{
				ProjectName: filepath.Base(abs),
				Summary:     "Zero-config workspace inspection",
			}, false, nil
		}
		return Manifest{}, false, fmt.Errorf("read claims manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, true, fmt.Errorf("parse claims manifest %s: %w", target, err)
	}
	if strings.TrimSpace(m.ProjectName) == "" {
		abs, _ := filepath.Abs(workDir)
		m.ProjectName = filepath.Base(abs)
	}
	for i := range m.Persistence {
		if m.Persistence[i].Method == "" {
			m.Persistence[i].Method = "POST"
		}
		if m.Persistence[i].ExpectMinDelta <= 0 {
			m.Persistence[i].ExpectMinDelta = 1
		}
		if m.Persistence[i].ExpectStatusMax <= 0 {
			m.Persistence[i].ExpectStatusMax = 299
		}
	}
	for i := range m.ProtectedRoutes {
		if m.ProtectedRoutes[i].Method == "" {
			m.ProtectedRoutes[i].Method = "GET"
		}
	}
	for i := range m.MoneyTampers {
		if m.MoneyTampers[i].Method == "" {
			m.MoneyTampers[i].Method = "POST"
		}
	}
	return m, true, nil
}

// WriteTemplate writes a starter .axoproof/claims.json so agents or builders have an immediate schema.
func WriteTemplate(workDir string) (string, error) {
	p := DefaultPath(workDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	abs, _ := filepath.Abs(workDir)
	sample := Manifest{
		ProjectName:  filepath.Base(abs),
		Summary:      "Describe in one plain sentence what feature was built for the business owner.",
		BaseURL:      "http://127.0.0.1:3000",
		HealthPath:   "/api/health",
		TouchesMoney: true,
		TouchesAuth:  true,
		Persistence: []PersistenceClaim{
			{
				Name:           "Create order saves a real row in orders table",
				Method:         "POST",
				Path:           "/api/orders",
				PayloadJSON:    `{"sku":"ITEM-01","quantity":1}`,
				SQLitePath:     "./app.db",
				Table:          "orders",
				ExpectMinDelta: 1,
			},
		},
		ProtectedRoutes: []ProtectedRouteClaim{
			{
				Name:   "Customer order list blocks unauthenticated strangers",
				Method: "GET",
				Path:   "/api/customers/orders",
			},
		},
		MoneyTampers: []MoneyTamperClaim{
			{
				Name:             "Checkout rejects negative quantity exploit",
				Method:           "POST",
				Path:             "/api/orders",
				TamperedBodyJSON: `{"sku":"ITEM-01","quantity":-3,"price":1}`,
				SQLitePath:       "./app.db",
				Table:            "orders",
			},
		},
		ExternalDeps: []ExternalBlindspot{
			{
				Service: "Payment Gateway Live Webhook",
				NoteEN:  "Complete one real small payment on staging/production to confirm bank settlement callbacks.",
				NoteID:  "Lakukan 1 transaksi kecil nyata di staging/produksi untuk memastikan callback pembayaran masuk.",
			},
		},
	}
	enc, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(p, append(enc, '\n'), 0o644); err != nil {
		return "", err
	}
	return p, nil
}
