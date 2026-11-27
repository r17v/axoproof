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
        <svg class="axo-emblem" viewBox="-36 -36 628 593" role="img" aria-label="AxoProof seal">
          <path fill="currentColor" fill-rule="evenodd" d="M 156.260 519.877 C 144.424 517.940 133.342 509.503 127.774 498.193 C 124.761 492.073 124.502 490.782 124.524 482.021 C 124.552 471.261 125.763 467.694 133.080 456.830 C 145.628 438.198 168.398 423.004 193.830 416.292 C 202.073 414.117 206.327 413.669 222.655 413.254 C 243.482 412.724 251.927 413.430 279.339 417.989 C 310.529 423.176 322.202 424.314 338.000 423.707 C 355.370 423.040 365.136 421.256 380.604 415.922 C 394.905 410.991 403.843 406.556 415.473 398.622 C 443.514 379.490 464.035 348.511 471.680 313.768 C 473.278 306.510 473.500 300.432 473.500 264.000 C 473.500 224.673 473.380 221.907 471.204 211.166 C 467.037 190.600 460.490 173.216 450.839 157.090 C 434.725 130.165 412.329 108.442 386.872 95.045 C 364.661 83.356 346.728 78.656 324.500 78.697 C 291.076 78.759 262.959 90.557 239.831 114.223 C 228.700 125.613 226.555 126.620 216.808 125.039 C 207.606 123.546 191.477 123.800 184.000 125.556 C 180.425 126.396 173.675 128.921 169.000 131.168 C 162.042 134.512 158.955 136.817 151.982 143.876 C 142.729 153.244 137.859 161.256 133.979 173.500 C 132.206 179.095 131.628 184.011 131.098 198.000 C 130.249 220.429 129.014 223.013 116.490 228.543 C 96.668 237.297 80.366 255.316 74.634 274.809 C 69.508 292.241 72.934 313.352 83.518 329.556 C 88.815 337.665 100.232 348.725 107.927 353.202 C 111.427 355.237 117.713 358.094 121.895 359.549 C 138.034 365.163 141.650 367.433 145.044 374.086 C 147.233 378.377 147.699 388.006 145.944 392.685 C 142.620 401.548 132.891 409.761 121.684 413.165 C 113.673 415.598 104.786 415.235 91.823 411.944 C 41.061 399.057 3.281 351.580 0.364 297.007 C -1.508 262.010 16.503 221.767 45.000 197.275 C 51.536 191.657 61.119 185.184 70.661 179.940 C 73.212 178.538 73.631 177.544 74.795 170.148 C 81.220 129.292 106.792 93.289 142.000 75.527 C 157.519 67.698 176.341 63.000 192.191 63.000 L 201.161 63.000 211.235 52.449 C 240.047 22.271 275.980 5.145 319.335 0.927 C 340.359 -1.118 368.732 2.412 392.500 10.029 C 424.249 20.204 451.376 37.300 478.065 63.954 C 523.518 109.348 549.359 163.087 555.025 224.000 C 555.736 231.644 555.995 276.237 555.797 357.000 L 555.500 478.500 552.276 485.089 C 537.582 515.116 492.521 515.140 477.741 485.128 C 474.507 478.561 474.499 478.507 474.000 459.561 L 473.500 440.575 466.566 446.863 C 445.853 465.646 412.666 481.985 382.134 488.431 C 353.423 494.493 328.367 494.053 288.000 486.775 C 268.705 483.297 263.363 482.703 250.500 482.609 C 232.632 482.479 223.624 484.194 211.346 490.065 C 204.248 493.459 202.046 495.210 194.346 503.585 C 189.481 508.877 183.700 514.248 181.500 515.521 C 174.558 519.539 164.645 521.250 156.260 519.877 Z M 281.000 400.694 C 237.750 384.211 199.193 347.153 182.985 306.488 C 172.388 279.902 170.032 263.528 170.039 216.500 C 170.045 180.183 170.181 179.424 177.923 172.798 C 182.424 168.945 188.243 167.000 195.267 167.000 C 202.599 167.000 217.892 163.999 229.500 160.282 C 245.163 155.267 260.751 147.345 273.182 138.082 C 287.424 127.468 289.162 126.633 297.000 126.633 C 304.543 126.633 306.970 127.689 318.500 135.984 C 342.596 153.321 370.695 164.138 398.950 166.954 C 405.247 167.582 411.874 168.795 413.676 169.651 C 418.240 171.816 423.765 178.231 424.517 182.237 C 424.863 184.083 424.961 203.347 424.734 225.047 C 424.285 267.947 424.031 270.257 417.470 290.941 C 403.323 335.540 371.494 372.676 329.241 393.879 C 307.075 405.003 296.299 406.525 281.000 400.694 Z M 304.240 366.634 C 321.647 360.082 334.954 351.248 349.500 336.587 C 368.136 317.804 378.400 299.614 385.673 272.482 C 388.240 262.908 388.366 261.249 388.760 231.870 L 389.171 201.239 379.919 199.539 C 354.226 194.817 329.775 184.623 306.295 168.842 C 302.332 166.179 298.395 164.000 297.545 164.000 C 296.695 164.000 291.837 166.876 286.750 170.391 C 263.722 186.302 241.592 195.157 211.000 200.701 L 206.500 201.516 206.562 228.508 C 206.632 258.906 207.661 267.253 213.482 284.651 C 220.559 305.802 230.189 321.379 245.905 337.095 C 259.867 351.057 275.380 361.271 291.500 367.115 C 297.768 369.388 296.830 369.423 304.240 366.634 Z M 277.500 314.087 C 273.275 312.112 244.728 286.579 241.757 282.118 C 239.570 278.835 239.429 270.005 241.499 266.002 C 243.533 262.068 249.056 258.027 253.351 257.330 C 260.610 256.152 263.547 257.570 273.505 267.056 C 278.669 271.975 283.468 276.000 284.171 276.000 C 284.874 276.000 296.711 264.788 310.475 251.085 C 331.978 229.676 336.098 226.011 339.750 225.046 C 344.798 223.712 350.308 224.881 354.495 228.175 C 357.693 230.691 360.946 237.267 360.978 241.282 C 361.008 245.034 358.627 250.484 355.756 253.240 C 354.461 254.483 340.073 268.631 323.783 284.680 C 299.952 308.159 293.416 314.068 290.332 314.925 C 284.943 316.421 282.097 316.235 277.500 314.087 Z"/>
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
