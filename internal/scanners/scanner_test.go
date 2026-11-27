package scanners_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/r17v/axoproof/internal/scanners"
)

func TestScanWorkspaceSeparatesFatalFromCustomerPolish(t *testing.T) {
	ws := t.TempDir()

	sabotagedOrderTS := `
export async function createOrder(req: any, res: any) {
  const unitPrice = parseFloat(req.body.price);
  return res.json({ ok: true, unitPrice });
}
`
	sabotagedUI := `
export function Page() {
  return (
    <div>
      <h1>Unleash the Power of Retail</h1>
      <p>Contact info@example.com</p>
      <div>Total Revenue: $42,500.00</div>
      <a href="#">Help</a>
      <button className="h-5" onClick={() => {}}>Save</button>
    </div>
  );
}
`
	if err := os.WriteFile(filepath.Join(ws, "order_controller.ts"), []byte(sabotagedOrderTS), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "page.tsx"), []byte(sabotagedUI), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := scanners.ScanWorkspace(ws)
	if err != nil {
		t.Fatalf("ScanWorkspace error: %v", err)
	}
	if res.FilesScanned != 2 {
		t.Fatalf("expected 2 scanned files, got %d", res.FilesScanned)
	}
	if len(res.FatalRisks) < 4 {
		t.Fatalf("expected at least 4 fatal risks (float money, client price, hardcoded revenue, dead button), got %d: %+v", len(res.FatalRisks), res.FatalRisks)
	}
	if len(res.PolishWarnings) < 4 {
		t.Fatalf("expected at least 4 customer-polish warnings (robotic copy, example.com, dead link, tiny button), got %d: %+v", len(res.PolishWarnings), res.PolishWarnings)
	}

	// Verify --scope diff filtering scans only the allowed file subset
	diffOnly, err := scanners.ScanWorkspaceFiltered(ws, map[string]bool{"order_controller.ts": true})
	if err != nil {
		t.Fatal(err)
	}
	if diffOnly.FilesScanned != 1 || len(diffOnly.PolishWarnings) != 0 {
		t.Fatalf("expected diff filter to scan only order_controller.ts (0 UI polish warnings), got %+v", diffOnly)
	}
}
