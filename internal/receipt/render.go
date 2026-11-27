package receipt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/r17v/axoproof/internal/model"
)

// ComputeSeal calculates a deterministic SHA-256 seal over all substantive receipt fields.
func ComputeSeal(r model.SealedReceipt) (string, error) {
	clone := r
	clone.SealSHA256 = ""
	canonical, err := json.Marshal(clone)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// VerifySeal checks whether a receipt's embedded SHA-256 seal matches its payload.
func VerifySeal(r model.SealedReceipt) bool {
	if len(r.SealSHA256) != 64 {
		return false
	}
	expected, err := ComputeSeal(r)
	if err != nil {
		return false
	}
	return expected == r.SealSHA256
}

// WriteArtifacts seals the receipt and writes both .axoproof/receipt.json and .axoproof/RECEIPT.html.
func WriteArtifacts(workDir string, r model.SealedReceipt, defaultLang string) (model.SealedReceipt, string, string, error) {
	if defaultLang != "id" {
		defaultLang = "en"
	}
	seal, err := ComputeSeal(r)
	if err != nil {
		return r, "", "", fmt.Errorf("compute receipt seal: %w", err)
	}
	r.SealSHA256 = seal

	outDir := filepath.Join(workDir, ".axoproof")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return r, "", "", err
	}

	jsonPath := filepath.Join(outDir, "receipt.json")
	jsonBytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return r, "", "", err
	}
	if err := os.WriteFile(jsonPath, append(jsonBytes, '\n'), 0o644); err != nil {
		return r, "", "", err
	}

	htmlPath := filepath.Join(outDir, "RECEIPT.html")
	htmlStr, err := RenderHTML(r, defaultLang)
	if err != nil {
		return r, jsonPath, "", err
	}
	if err := os.WriteFile(htmlPath, []byte(htmlStr), 0o644); err != nil {
		return r, jsonPath, "", err
	}

	return r, jsonPath, htmlPath, nil
}

type viewData struct {
	StampCSS    template.CSS
	Receipt     model.SealedReceipt
	DefaultLang string
	ToneClass   string
	ShortSeal   string
	HasRepairs  bool
}

// RenderHTML produces a self-contained Swiss archival inspection dossier (zero external requests, zero AI card-grid cliches).
func RenderHTML(r model.SealedReceipt, defaultLang string) (string, error) {
	if defaultLang != "id" {
		defaultLang = "en"
	}
	toneClass := "tone-warn"
	switch r.Verdict {
	case model.VerdictSafeToTest:
		toneClass = "tone-pass"
	case model.VerdictHoldRelease:
		toneClass = "tone-fail"
	}
	shortSeal := r.SealSHA256
	if len(shortSeal) > 16 {
		shortSeal = shortSeal[:16]
	}

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}

	tpl, err := template.New("receipt").Funcs(funcMap).Parse(receiptHTMLTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, viewData{
		StampCSS: template.CSS("/* Hallmark - checked: 2026-11-27 - genre: editorial - shape: app screen\n" +
			" * critique: P5 H5 E5 S5 R5 V5 - cut nested dashboard cards and dark neon boxes in favor of a warm archival inspection dossier with asymmetric two-column ledger rules and real tabular measurements\n" +
			" * built: AxoProof - v0.1.0 */"),
		Receipt:     r,
		DefaultLang: defaultLang,
		ToneClass:   toneClass,
		ShortSeal:   shortSeal,
		HasRepairs:  len(r.FatalRisks) > 0 || len(r.PolishWarnings) > 0,
	})
	return buf.String(), err
}

const receiptHTMLTemplate = `<!DOCTYPE html>
<html lang="{{.DefaultLang}}" data-lang="{{.DefaultLang}}" data-surface="paper">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="Independent runtime proof, money safety, and human-readiness inspection ledger sealed by AxoProof.">
  <title>AxoProof Inspection Dossier: {{.Receipt.ProjectName}}</title>
  <style>{{.StampCSS}}
    :root {
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
      --space-12: 48px;
      --space-16: 64px;
    }

    html[data-surface="night"] {
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
    }

    * { box-sizing: border-box; }
    html, body {
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
    }

    .sheet {
      max-width: 1040px;
      margin: 0 auto;
      padding: var(--space-8) var(--space-6) var(--space-16);
    }

    /* Masthead */
    .masthead {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: space-between;
      gap: var(--space-4);
      padding-bottom: var(--space-5);
      border-bottom: 2px solid var(--rule-strong);
    }
    .identity {
      display: flex;
      align-items: center;
      gap: var(--space-3);
      min-width: 0;
    }
    .axo-emblem {
      width: 32px;
      height: 32px;
      flex-shrink: 0;
      color: var(--brand);
    }
    .doc-name {
      margin: 0;
      font-size: 17px;
      font-weight: 700;
      color: var(--ink);
    }
    .doc-sub {
      margin: 0;
      font-size: 14px;
      color: var(--ink-3);
    }
    .toolbar {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: var(--space-2);
    }
    .seg-group {
      display: inline-flex;
      border: 1px solid var(--rule);
      border-radius: 6px;
      padding: 2px;
      background: var(--paper-inset);
    }
    .ctrl-btn {
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
    }
    .ctrl-btn:hover {
      color: var(--ink);
    }
    .ctrl-btn:active {
      background: var(--rule);
      color: var(--ink);
    }
    .ctrl-btn:disabled {
      opacity: 0.4;
      cursor: not-allowed;
    }
    .ctrl-btn[aria-pressed="true"] {
      background: var(--paper);
      color: var(--ink);
      border-color: var(--rule);
    }
    .ctrl-btn:focus-visible,
    .copy-action:focus-visible,
    summary:focus-visible {
      outline: 2px solid var(--brand);
      outline-offset: 2px;
    }

    /* Asymmetric Editorial Verdict Spread */
    .verdict-spread {
      display: grid;
      grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
      gap: var(--space-10);
      padding: var(--space-10) 0;
      border-bottom: 1px solid var(--rule-strong);
      align-items: start;
    }
    .verdict-head {
      margin: 0 0 var(--space-4) 0;
      font-family: var(--font-display);
      font-size: 36px;
      font-weight: 700;
      line-height: 1.12;
      color: var(--ink);
      text-wrap: balance;
      overflow-wrap: break-word;
      min-width: 0;
    }
    .tone-pass .verdict-head { color: var(--pass-ink); }
    .tone-warn .verdict-head { color: var(--warn-ink); }
    .tone-fail .verdict-head { color: var(--fail-ink); }

    .verdict-lede {
      margin: 0;
      max-width: 58ch;
      font-size: 17px;
      color: var(--ink-2);
    }
    .copy-action {
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
    }
    .copy-action:hover {
      background: var(--brand-press);
      border-color: var(--brand-press);
    }
    .copy-action:active {
      transform: translateY(1px);
    }
    .copy-action:disabled {
      opacity: 0.45;
      cursor: not-allowed;
    }

    /* Right-column Tabular Summary Ledger */
    .summary-ledger {
      margin: 0;
      border-top: 1px solid var(--rule);
      font-variant-numeric: tabular-nums;
    }
    .ledger-pair {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      gap: var(--space-4);
      padding: var(--space-3) 0;
      border-bottom: 1px solid var(--rule);
    }
    .ledger-pair dt {
      font-size: 14px;
      color: var(--ink-2);
    }
    .ledger-pair dd {
      margin: 0;
      font-size: 15px;
      font-weight: 700;
      color: var(--ink);
      text-align: right;
    }
    .val-fail { color: var(--fail-ink); }
    .val-pass { color: var(--pass-ink); }

    /* Two-Column Swiss Inspection Chapters (Zero nested cards) */
    .chapter {
      display: grid;
      grid-template-columns: 240px minmax(0, 1fr);
      gap: var(--space-8);
      padding: var(--space-8) 0;
      border-bottom: 1px solid var(--rule);
    }
    .chapter-head h2 {
      margin: 0 0 var(--space-2) 0;
      font-family: var(--font-display);
      font-size: 21px;
      font-weight: 700;
      line-height: 1.25;
      color: var(--ink);
      text-wrap: balance;
      overflow-wrap: break-word;
      min-width: 0;
    }
    .chapter-note {
      margin: 0;
      font-size: 14px;
      color: var(--ink-3);
      max-width: 32ch;
    }

    /* Ruled Ledger Rows */
    .finding-list {
      list-style: none;
      margin: 0;
      padding: 0;
      border-top: 1px solid var(--rule);
    }
    .finding-row {
      padding: var(--space-4) 0;
      border-bottom: 1px solid var(--rule);
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: var(--space-4);
      align-items: baseline;
    }
    .finding-title {
      margin: 0 0 var(--space-1) 0;
      font-size: 16px;
      font-weight: 700;
      color: var(--ink);
    }
    .finding-title .flag-fail {
      color: var(--fail-ink);
      margin-right: var(--space-2);
    }
    .finding-title .flag-warn {
      color: var(--warn-ink);
      margin-right: var(--space-2);
    }
    .finding-impact {
      margin: 0;
      max-width: 64ch;
      font-size: 15px;
      color: var(--ink-2);
    }
    .finding-loc {
      font-family: var(--font-mono);
      font-size: 13px;
      color: var(--ink-3);
      white-space: nowrap;
    }

    .clean-line {
      margin: 0;
      padding: var(--space-4) 0;
      border-top: 1px solid var(--rule);
      border-bottom: 1px solid var(--rule);
      font-size: 15px;
      font-weight: 600;
      color: var(--pass-ink);
    }

    /* Real Measurement Table for Live Probes */
    .probe-table-wrap {
      width: 100%;
      overflow-x: auto;
    }
    table.probe-table {
      width: 100%;
      border-collapse: collapse;
      border-top: 1px solid var(--rule);
      font-variant-numeric: tabular-nums;
    }
    .probe-table th {
      text-align: left;
      font-size: 13px;
      font-weight: 600;
      color: var(--ink-3);
      padding: var(--space-2) var(--space-3) var(--space-2) 0;
      border-bottom: 1px solid var(--rule);
    }
    .probe-table td {
      padding: var(--space-3) var(--space-3) var(--space-3) 0;
      border-bottom: 1px solid var(--rule);
      vertical-align: top;
      font-size: 15px;
    }
    .probe-table td.probe-code {
      font-family: var(--font-mono);
      font-size: 13px;
      color: var(--ink-2);
      white-space: nowrap;
    }
    .status-word-pass { color: var(--pass-ink); font-weight: 700; }
    .status-word-fail { color: var(--fail-ink); font-weight: 700; }
    .status-word-skip { color: var(--warn-ink); font-weight: 700; }

    /* Perforated Tear-Off Agent Slip */
    .agent-slip {
      margin-top: var(--space-8);
      padding-top: var(--space-4);
      border-top: 1px dashed var(--rule-strong);
    }
    .agent-slip summary {
      min-height: 44px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      cursor: pointer;
      font-size: 15px;
      font-weight: 700;
      color: var(--ink-2);
    }
    pre.repair-brief {
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
    }

    /* Notary Footer */
    .notary-foot {
      margin-top: var(--space-8);
      padding-top: var(--space-4);
      border-top: 1px solid var(--rule);
      display: flex;
      flex-wrap: wrap;
      justify-content: space-between;
      gap: var(--space-3);
      font-size: 13px;
      color: var(--ink-3);
    }
    .notary-foot code {
      font-family: var(--font-mono);
      color: var(--ink-2);
    }

    @media (max-width: 760px) {
      .verdict-spread,
      .chapter {
        grid-template-columns: minmax(0, 1fr);
        gap: var(--space-5);
      }
      .finding-row {
        grid-template-columns: minmax(0, 1fr);
        gap: var(--space-1);
      }
      .verdict-head {
        font-size: 28px;
      }
    }

    html[data-lang="en"] .lang-id { display: none; }
    html[data-lang="id"] .lang-en { display: none; }
  </style>
</head>
<body class="{{.ToneClass}}">
  <main class="sheet">
    <header class="masthead">
      <div class="identity">
        <svg class="axo-emblem" viewBox="-39.76 -39.76 725.52 647.52" role="img" aria-label="AxoProof seal">
          <path fill="currentColor" fill-rule="evenodd" d="M 192.1 566.9 C 179.3 564.4 167.7 555.4 161.9 543.4 C 158.7 536.8 158.5 536 158.5 525.4 C 158.5 514.3 158.5 514.3 163.2 504.9 C 184.2 462.5 229.4 430.5 281.8 420.8 C 300.1 417.4 336.1 417.6 355.5 421.3 C 363.2 422.7 373.1 424.6 377.6 425.4 C 451.2 438.9 509.8 412.8 539.6 353.2 C 578 276.6 557.2 173.3 491.7 115.9 C 444 74.1 383.7 60.1 335.1 79.6 C 318.2 86.3 302.7 98 289.9 113.7 C 282.5 122.7 282.3 122.7 268.7 119.4 C 219.3 107.4 171.9 134.1 160 180.5 C 158.2 187.2 157.9 191 158.2 203.8 C 158.5 222.2 158.8 221.7 144.7 225.4 C 105.9 235.6 77.9 262.5 66.4 300.5 C 62.6 313.1 62.3 336.2 65.8 348.3 C 76.7 385.9 110.1 411.7 151.7 414.6 C 169.1 415.9 169.7 422.1 154.3 440.6 C 142.7 454.5 129.6 460.5 111 460.4 C 68.3 460.2 24.1 424.2 8.2 376.5 C -4.4 338.7 -1.2 297.3 17.2 260.5 C 32.7 229.7 61.2 200.6 90.1 186 C 95.7 183.1 95.9 182.9 96.5 178.3 C 106.5 107 167.3 55 240.7 55 L 253.2 55 261.8 47 C 304 7.9 362.3 -7.3 427 4 C 542.6 24.1 631.6 122.8 644.5 245.2 C 646.8 267.7 646.8 525 644.4 533.1 C 630.7 579.7 560.1 579.5 546.6 532.9 C 545.3 528.4 544.9 521.4 544.7 496.3 L 544.5 465.2 540.5 468.5 C 490.4 510.8 432.8 524 356 510.8 C 299.7 501 263.6 512 241.5 545.6 C 229.6 563.7 212.7 571 192.1 566.9 Z"/>
        </svg>
        <div>
          <p class="doc-name">AxoProof <span class="lang-en">Inspection Record</span><span class="lang-id">Lembar Pemeriksaan Hasil Kerja AI</span></p>
          <p class="doc-sub">{{.Receipt.ProjectName}} ({{.Receipt.GeneratedAtUTC}})</p>
        </div>
      </div>

      <div class="toolbar">
        <div class="seg-group" role="group" aria-label="Language">
          <button type="button" class="ctrl-btn" data-set-lang="en" aria-pressed="{{if eq .DefaultLang "en"}}true{{else}}false{{end}}">EN</button>
          <button type="button" class="ctrl-btn" data-set-lang="id" aria-pressed="{{if eq .DefaultLang "id"}}true{{else}}false{{end}}">ID</button>
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
          <span class="lang-en">{{.Receipt.HeadlineEN}}</span>
          <span class="lang-id">{{.Receipt.HeadlineID}}</span>
        </h1>
        <p class="verdict-lede">
          <span class="lang-en">{{.Receipt.SubheadlineEN}}</span>
          <span class="lang-id">{{.Receipt.SubheadlineID}}</span>
        </p>
        {{if .HasRepairs}}
          <button type="button" class="copy-action" id="copy-repair-btn">
            <span class="lang-en">Copy AI repair brief</span>
            <span class="lang-id">Salin perintah untuk AI</span>
          </button>
        {{end}}
      </div>

      <dl class="summary-ledger">
        <div class="ledger-pair">
          <dt><span class="lang-en">Release status</span><span class="lang-id">Status serah terima</span></dt>
          <dd class="{{if eq .ToneClass "tone-pass"}}val-pass{{else if eq .ToneClass "tone-fail"}}val-fail{{end}}">
            {{if eq .ToneClass "tone-pass"}}
              <span class="lang-en">Verified live</span><span class="lang-id">Terbukti jalan</span>
            {{else if eq .ToneClass "tone-fail"}}
              <span class="lang-en">Hold release</span><span class="lang-id">Tahan rilis</span>
            {{else}}
              <span class="lang-en">Pending live check</span><span class="lang-id">Menunggu uji jalan</span>
            {{end}}
          </dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Source files inspected</span><span class="lang-id">Berkas kode diperiksa</span></dt>
          <dd>{{.Receipt.FilesScanned}}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Blocking business or money defects</span><span class="lang-id">Masalah fatal bisnis &amp; uang</span></dt>
          <dd class="{{if .Receipt.FatalRisks}}val-fail{{else}}val-pass{{end}}">{{len .Receipt.FatalRisks}}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Customer screen &amp; language notes</span><span class="lang-id">Catatan tampilan &amp; bahasa pelanggan</span></dt>
          <dd>{{len .Receipt.PolishWarnings}}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Live HTTP &amp; database probes</span><span class="lang-id">Uji database &amp; pintu secara live</span></dt>
          <dd>{{len .Receipt.RuntimeProbes}}</dd>
        </div>
        <div class="ledger-pair">
          <dt><span class="lang-en">Axoloom memory link</span><span class="lang-id">Jembatan memori Axoloom</span></dt>
          <dd>
            {{if .Receipt.Axoloom.Connected}}
              <span class="lang-en">Connected ({{.Receipt.Axoloom.IndexedEndpoints}} routes, {{.Receipt.Axoloom.IndexedSQLTables}} tables)</span>
              <span class="lang-id">Terhubung ({{.Receipt.Axoloom.IndexedEndpoints}} route, {{.Receipt.Axoloom.IndexedSQLTables}} tabel)</span>
            {{else}}
              <span class="lang-en">Standalone</span>
              <span class="lang-id">Mandiri</span>
            {{end}}
          </dd>
        </div>
      </dl>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">01. Money, Privacy &amp; Real Wiring</span><span class="lang-id">01. Kasir, Kunci Data &amp; Fitur Asli</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Defects that lose orders, miscalculate currency, or leave private records open.</span>
          <span class="lang-id">Masalah yang bikin data pesanan hilang, salah hitung Rupiah, atau data pelanggan bocor.</span>
        </p>
      </div>
      <div>
        {{if .Receipt.FatalRisks}}
          <ol class="finding-list">
            {{range $idx, $f := .Receipt.FatalRisks}}
              <li class="finding-row">
                <div>
                  <p class="finding-title">
                    <span class="flag-fail">{{inc $idx}}.</span>
                    <span class="lang-en">{{$f.TitleEN}}</span>
                    <span class="lang-id">{{$f.TitleID}}</span>
                  </p>
                  <p class="finding-impact">
                    <span class="lang-en">{{$f.BusinessImpactEN}}</span>
                    <span class="lang-id">{{$f.BusinessImpactID}}</span>
                  </p>
                </div>
                {{if $f.File}}<span class="finding-loc">{{$f.File}}{{if $f.Line}}:{{$f.Line}}{{end}}</span>{{end}}
              </li>
            {{end}}
          </ol>
        {{else}}
          <p class="clean-line">
            <span class="lang-en">Passed: no fake buttons, floating-point currency bugs, or unlocked endpoints detected.</span>
            <span class="lang-id">Bersih: tidak ditemukan tombol pajangan kosong, salah tipe uang, maupun pintu data terbuka.</span>
          </p>
        {{end}}
      </div>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">02. Live Server &amp; Database Ledger</span><span class="lang-id">02. Bukti Kirim Data &amp; Isi Database</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Measured directly by AxoProof against your running app and SQLite tables.</span>
          <span class="lang-id">Diukur langsung oleh AxoProof ke server dan tabel database nyata, bukan pengakuan AI.</span>
        </p>
      </div>
      <div class="probe-table-wrap">
        {{if .Receipt.RuntimeProbes}}
          <table class="probe-table">
            <thead>
              <tr>
                <th><span class="lang-en">Check</span><span class="lang-id">Pengujian</span></th>
                <th><span class="lang-en">Outcome</span><span class="lang-id">Hasil Nyata</span></th>
                <th><span class="lang-en">Measured Proof</span><span class="lang-id">Bukti Mesin</span></th>
              </tr>
            </thead>
            <tbody>
              {{range .Receipt.RuntimeProbes}}
                <tr>
                  <td>
                    {{if and .Executed .Passed}}
                      <span class="status-word-pass"><span class="lang-en">Proven</span><span class="lang-id">Terbukti</span></span>
                    {{else if .Executed}}
                      <span class="status-word-fail"><span class="lang-en">Failed</span><span class="lang-id">Gagal</span></span>
                    {{else}}
                      <span class="status-word-skip"><span class="lang-en">Pending</span><span class="lang-id">Tertunda</span></span>
                    {{end}}
                    <div>{{.Name}}</div>
                  </td>
                  <td>
                    <span class="lang-en">{{.SummaryEN}}</span>
                    <span class="lang-id">{{.SummaryID}}</span>
                  </td>
                  <td class="probe-code">{{.RawEvidence}}</td>
                </tr>
              {{end}}
            </tbody>
          </table>
        {{else}}
          <p class="finding-impact">
            <span class="lang-en">No live server URL was configured in .axoproof/claims.json yet. Static checks completed; live database write counts remain unmeasured.</span>
            <span class="lang-id">Alamat server lokal belum diisi di .axoproof/claims.json. Kode sudah diperiksa, namun pertambahan baris database belum diukur secara live.</span>
          </p>
        {{end}}
      </div>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">03. Customer Screen &amp; Human Copy</span><span class="lang-id">03. Kerapian Layar &amp; Bahasa Manusia</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Checks for leftover sample contacts, dead links, robotic cliches, and cramped mobile buttons.</span>
          <span class="lang-id">Memeriksa sisa email contoh palsu, link buntu, bahasa brosur robotik, dan tombol HP kekecilan.</span>
        </p>
      </div>
      <div>
        {{if .Receipt.PolishWarnings}}
          <ol class="finding-list">
            {{range $idx, $w := .Receipt.PolishWarnings}}
              <li class="finding-row">
                <div>
                  <p class="finding-title">
                    <span class="flag-warn">{{inc $idx}}.</span>
                    <span class="lang-en">{{$w.TitleEN}}</span>
                    <span class="lang-id">{{$w.TitleID}}</span>
                  </p>
                  <p class="finding-impact">
                    <span class="lang-en">{{$w.BusinessImpactEN}}</span>
                    <span class="lang-id">{{$w.BusinessImpactID}}</span>
                  </p>
                </div>
                {{if $w.File}}<span class="finding-loc">{{$w.File}}{{if $w.Line}}:{{$w.Line}}{{end}}</span>{{end}}
              </li>
            {{end}}
          </ol>
        {{else}}
          <p class="clean-line">
            <span class="lang-en">Clean: no fake sample contacts, dead '#' anchors, or robotic brochure copy detected.</span>
            <span class="lang-id">Rapi: tidak ada kontak contoh palsu, link '#' buntu, maupun kalimat promosi robotik.</span>
          </p>
        {{end}}
      </div>
    </section>

    <section class="chapter">
      <div class="chapter-head">
        <h2><span class="lang-en">04. Owner 2-Minute Check</span><span class="lang-id">04. Cek Mata Owner 2 Menit</span></h2>
        <p class="chapter-note">
          <span class="lang-en">Real-world items outside local automation that still need your own eyes before launch.</span>
          <span class="lang-id">Hal dunia nyata di luar jangkauan mesin lokal yang tetap wajib Anda pastikan sendiri.</span>
        </p>
      </div>
      <ol class="finding-list">
        {{range $idx, $b := .Receipt.Blindspots}}
          <li class="finding-row">
            <div>
              <p class="finding-title">{{inc $idx}}. <span class="lang-en">{{$b.TitleEN}}</span><span class="lang-id">{{$b.TitleID}}</span></p>
              <p class="finding-impact"><span class="lang-en">{{$b.WhyEN}}</span><span class="lang-id">{{$b.WhyID}}</span></p>
            </div>
          </li>
        {{end}}
      </ol>
    </section>

    <details class="agent-slip">
      <summary>
        <span>
          <span class="lang-en">Technical repair slip for AI coding agent ({{len .Receipt.FatalRisks}} blocking, {{len .Receipt.PolishWarnings}} polish)</span>
          <span class="lang-id">Slip instruksi teknis untuk AI Coding Agent ({{len .Receipt.FatalRisks}} fatal, {{len .Receipt.PolishWarnings}} polesan)</span>
        </span>
      </summary>
      <pre class="repair-brief" id="repair-brief-text">{{if or .Receipt.FatalRisks .Receipt.PolishWarnings}}Fix every defect verified by AxoProof below, then re-run: axoproof verify --sync-axoloom

{{range .Receipt.FatalRisks}}[BLOCKING / {{.ID}}] {{if .File}}{{.File}}:{{.Line}} - {{end}}{{.RepairInstruction}}
{{end}}{{range .Receipt.PolishWarnings}}[POLISH / {{.ID}}] {{if .File}}{{.File}}:{{.Line}} - {{end}}{{.RepairInstruction}}
{{end}}{{else}}No repair actions pending.{{end}}</pre>
    </details>

    <footer class="notary-foot">
      <span>Git state: <code>{{.Receipt.GitCommit}}</code> ({{.Receipt.GitDirtyFiles}} uncommitted)</span>
      <span>SHA-256 seal: <code>{{.Receipt.SealSHA256}}</code></span>
    </footer>
  </main>

  <script>
    (function () {
      var html = document.documentElement;
      var langBtns = document.querySelectorAll("[data-set-lang]");
      for (var i = 0; i < langBtns.length; i++) {
        langBtns[i].addEventListener("click", function () {
          var target = this.getAttribute("data-set-lang");
          html.setAttribute("data-lang", target);
          html.setAttribute("lang", target);
          for (var j = 0; j < langBtns.length; j++) {
            langBtns[j].setAttribute("aria-pressed", langBtns[j].getAttribute("data-set-lang") === target ? "true" : "false");
          }
        });
      }

      var surfBtns = document.querySelectorAll("[data-set-surface]");
      for (var k = 0; k < surfBtns.length; k++) {
        surfBtns[k].addEventListener("click", function () {
          var surf = this.getAttribute("data-set-surface");
          html.setAttribute("data-surface", surf);
          for (var m = 0; m < surfBtns.length; m++) {
            surfBtns[m].setAttribute("aria-pressed", surfBtns[m].getAttribute("data-set-surface") === surf ? "true" : "false");
          }
        });
      }

      var copyBtn = document.getElementById("copy-repair-btn");
      var briefEl = document.getElementById("repair-brief-text");
      if (copyBtn && briefEl && navigator.clipboard) {
        copyBtn.addEventListener("click", function () {
          var isID = html.getAttribute("data-lang") === "id";
          navigator.clipboard.writeText(briefEl.textContent || "").then(function () {
            copyBtn.textContent = isID ? "Berhasil disalin! Tempel ke AI Agent" : "Copied! Paste into your AI agent";
          });
        });
      }
    })();
  </script>
</body>
</html>
`
