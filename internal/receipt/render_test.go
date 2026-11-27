package receipt_test

import (
	"strings"
	"testing"

	"github.com/r17v/axoproof/internal/model"
	"github.com/r17v/axoproof/internal/receipt"
)

func TestReceiptSealDetectsTamperingAndRendersBilingualHTML(t *testing.T) {
	ws := t.TempDir()
	rec := model.SealedReceipt{
		SchemaVersion:  "1.0",
		ProjectName:    "Acme Roastery",
		GeneratedAtUTC: "2026-11-27T10:00:00Z",
		GitCommit:      "abc1234",
		Verdict:        model.VerdictSafeToTest,
		HeadlineEN:     "SAFE TO TEST WITH REAL CUSTOMERS",
		HeadlineID:     "TERBUKTI JALAN & AMAN DITES KE PELANGGAN",
		SubheadlineEN:  "All checks passed.",
		SubheadlineID:  "Semua pemeriksaan lulus.",
		FilesScanned:   8,
	}

	sealed, _, htmlPath, err := receipt.WriteArtifacts(ws, rec, "id")
	if err != nil {
		t.Fatalf("WriteArtifacts error: %v", err)
	}
	if !receipt.VerifySeal(sealed) {
		t.Fatalf("expected freshly sealed receipt to pass VerifySeal")
	}

	// Simulate an AI agent trying to forge Verdict from HOLD_RELEASE to SAFE_TO_TEST
	forged := sealed
	forged.FilesScanned = 999
	if receipt.VerifySeal(forged) {
		t.Fatalf("expected tampered receipt to fail VerifySeal")
	}

	htmlStr, err := receipt.RenderHTML(sealed, "id")
	if err != nil {
		t.Fatalf("RenderHTML error: %v", err)
	}
	if htmlPath == "" || !strings.Contains(htmlStr, `data-lang="id"`) || !strings.Contains(htmlStr, "TERBUKTI JALAN") {
		t.Fatalf("expected bilingual HTML output containing Indonesian headline")
	}
}
