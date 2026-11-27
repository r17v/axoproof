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
	BannerClass string
	ShortSeal   string
}

// RenderHTML produces a self-contained, zero-CDN, WCAG-AA compliant bilingual HTML receipt.
func RenderHTML(r model.SealedReceipt, defaultLang string) (string, error) {
	if defaultLang != "id" {
		defaultLang = "en"
	}
	bannerClass := "banner-amber"
	switch r.Verdict {
	case model.VerdictSafeToTest:
		bannerClass = "banner-green"
	case model.VerdictHoldRelease:
		bannerClass = "banner-red"
	}
	shortSeal := r.SealSHA256
	if len(shortSeal) > 16 {
		shortSeal = shortSeal[:16]
	}

	tpl, err := template.New("receipt").Parse(receiptHTMLTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, viewData{
		StampCSS:    template.CSS("/* Hallmark - checked: 2026-11-27 - genre: modern-minimal - shape: app screen\n * critique: strict 4px spatial rhythm inspection ledger, synaptic violet brand token, hairline status cards, and bilingual EN/ID owner toggle\n * built: AxoProof - v0.1.0 */"),
		Receipt:     r,
		DefaultLang: defaultLang,
		BannerClass: bannerClass,
		ShortSeal:   shortSeal,
	})
	return buf.String(), err
}

const receiptHTMLTemplate = `<!DOCTYPE html>
<html lang="{{.DefaultLang}}" data-lang="{{.DefaultLang}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="Independent runtime proof, money safety, and human-readiness receipt sealed by AxoProof.">
  <title>AxoProof Receipt: {{.Receipt.ProjectName}}</title>
  <style>{{.StampCSS}}
    :root {
      --bg: #0f0e17;
      --surface: #181624;
      --surface-2: #211e30;
      --code-bg: #0b0a10;
      --code-text: #d9d4f0;
      --border: #2e2a44;
      --text: #f3f1fb;
      --muted: #b8b2cf;
      --brand: #5c45b9;
      --brand-active: #4c37a0;
      --on-brand: #f7f5ff;
      --accent: #8b78f6;
      --green-bg: #07291e;
      --green-border: #10b981;
      --green-text: #a7f3d0;
      --amber-bg: #2c1d08;
      --amber-border: #f59e0b;
      --amber-text: #fde68a;
      --red-bg: #2e0d16;
      --red-border: #f43f5e;
      --red-text: #fecdd3;
      --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      --font-mono: ui-monospace, SFMono-Regular, Menlo, monospace;
    }
    * { box-sizing: border-box; }
    html, body {
      margin: 0;
      padding: 0;
      background: var(--bg);
      color: var(--text);
      font-family: var(--font-sans);
      line-height: 1.55;
      overflow-x: clip;
    }
    .page {
      max-width: 920px;
      margin: 0 auto;
      padding: 28px 20px 56px;
    }
    .topbar {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--border);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .brand-mark {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      background: var(--brand);
      color: var(--on-brand);
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-weight: 800;
      font-size: 18px;
    }
    .brand-title {
      margin: 0;
      font-size: 20px;
      font-weight: 700;
    }
    .brand-sub {
      margin: 0;
      font-size: 14px;
      color: var(--muted);
    }
    .lang-switch {
      display: inline-flex;
      gap: 4px;
      background: var(--surface);
      padding: 4px;
      border-radius: 12px;
      border: 1px solid var(--border);
    }
    .lang-btn {
      min-height: 44px;
      min-width: 56px;
      padding: 0 16px;
      border: 1px solid transparent;
      border-radius: 8px;
      background: transparent;
      color: var(--muted);
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
    }
    .lang-btn:focus-visible, summary:focus-visible {
      outline: 2px solid var(--accent);
      outline-offset: 2px;
    }
    .lang-btn:active {
      background: var(--brand-active);
      color: var(--on-brand);
    }
    .lang-btn:disabled {
      opacity: 0.45;
      cursor: not-allowed;
    }
    .lang-btn[aria-pressed="true"] {
      background: var(--brand);
      color: var(--on-brand);
    }
    .verdict-banner {
      margin-top: 24px;
      padding: 24px;
      border-radius: 12px;
      border: 2px solid var(--border);
      color: var(--text);
    }
    .banner-green { background: var(--green-bg); border-color: var(--green-border); }
    .banner-amber { background: var(--amber-bg); border-color: var(--amber-border); }
    .banner-red   { background: var(--red-bg);   border-color: var(--red-border); }
    .verdict-title {
      margin: 0 0 8px 0;
      font-size: 24px;
      font-weight: 800;
      text-wrap: balance;
    }
    .banner-green .verdict-title { color: var(--green-text); }
    .banner-amber .verdict-title { color: var(--amber-text); }
    .banner-red   .verdict-title { color: var(--red-text); }
    .verdict-sub {
      margin: 0;
      font-size: 16px;
      color: var(--text);
    }
    .meta-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(192px, 1fr));
      gap: 12px;
      margin-top: 16px;
    }
    .meta-card {
      background: var(--surface);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 16px;
    }
    .meta-label {
      font-size: 13px;
      color: var(--muted);
      margin: 0 0 4px 0;
    }
    .meta-val {
      font-size: 16px;
      font-weight: 700;
      margin: 0;
      word-break: break-word;
    }
    .section {
      margin-top: 28px;
      background: var(--surface);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px 24px;
    }
    .section h2 {
      margin: 0 0 8px 0;
      font-size: 19px;
      text-wrap: balance;
    }
    .section-desc {
      margin: 0 0 16px 0;
      font-size: 14px;
      color: var(--muted);
    }
    .item-list {
      list-style: none;
      margin: 0;
      padding: 0;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .item-card {
      background: var(--surface-2);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
    }
    .item-card.fatal { border-color: var(--red-border); }
    .item-card.polish { border-color: var(--amber-border); }
    .item-card.pass { border-color: var(--green-border); }
    .item-title {
      margin: 0 0 4px 0;
      font-size: 16px;
      font-weight: 700;
    }
    .item-impact {
      margin: 0;
      font-size: 15px;
      color: var(--text);
    }
    .item-meta {
      margin: 8px 0 0 0;
      font-family: var(--font-mono);
      font-size: 13px;
      color: var(--muted);
    }
    .empty-ok {
      margin: 0;
      padding: 16px;
      border-radius: 8px;
      background: var(--green-bg);
      border: 1px solid var(--green-border);
      color: var(--green-text);
      font-weight: 600;
    }
    details.repair-drawer {
      margin-top: 28px;
      background: var(--surface);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 8px 20px;
    }
    details.repair-drawer summary {
      min-height: 44px;
      display: flex;
      align-items: center;
      cursor: pointer;
      font-weight: 700;
      font-size: 16px;
    }
    pre.repair-code {
      background: var(--code-bg);
      color: var(--code-text);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px 16px;
      overflow-x: auto;
      font-family: var(--font-mono);
      font-size: 13px;
    }
    .seal-footer {
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid var(--border);
      font-size: 13px;
      color: var(--muted);
      display: flex;
      flex-wrap: wrap;
      justify-content: space-between;
      gap: 8px;
    }
    html[data-lang="en"] .lang-id { display: none; }
    html[data-lang="id"] .lang-en { display: none; }
  </style>
</head>
<body>
  <main class="page">
    <header class="topbar">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true">A</span>
        <div>
          <p class="brand-title">AxoProof <span class="lang-en">Owner Verification Receipt</span><span class="lang-id">Struk Pembuktian Hasil Kerja AI</span></p>
          <p class="brand-sub">{{.Receipt.ProjectName}} ({{.Receipt.GeneratedAtUTC}})</p>
        </div>
      </div>
      <div class="lang-switch" role="group" aria-label="Receipt language">
        <button type="button" class="lang-btn" data-set-lang="en" aria-pressed="{{if eq .DefaultLang "en"}}true{{else}}false{{end}}">EN</button>
        <button type="button" class="lang-btn" data-set-lang="id" aria-pressed="{{if eq .DefaultLang "id"}}true{{else}}false{{end}}">ID</button>
      </div>
    </header>

    <section class="verdict-banner {{.BannerClass}}" aria-live="polite">
      <h1 class="verdict-title">
        <span class="lang-en">{{.Receipt.HeadlineEN}}</span>
        <span class="lang-id">{{.Receipt.HeadlineID}}</span>
      </h1>
      <p class="verdict-sub">
        <span class="lang-en">{{.Receipt.SubheadlineEN}}</span>
        <span class="lang-id">{{.Receipt.SubheadlineID}}</span>
      </p>
    </section>

    <section class="meta-grid" aria-label="Inspection summary">
      <div class="meta-card">
        <p class="meta-label"><span class="lang-en">Files Inspected</span><span class="lang-id">Berkas Diperiksa</span></p>
        <p class="meta-val">{{.Receipt.FilesScanned}}</p>
      </div>
      <div class="meta-card">
        <p class="meta-label"><span class="lang-en">Fatal Business Risks</span><span class="lang-id">Risiko Fatal Bisnis</span></p>
        <p class="meta-val">{{len .Receipt.FatalRisks}}</p>
      </div>
      <div class="meta-card">
        <p class="meta-label"><span class="lang-en">Customer-Ready Polish</span><span class="lang-id">Polesan Layar Pelanggan</span></p>
        <p class="meta-val">{{len .Receipt.PolishWarnings}}</p>
      </div>
      <div class="meta-card">
        <p class="meta-label"><span class="lang-en">Axoloom Brain Link</span><span class="lang-id">Koneksi Otak Axoloom</span></p>
        <p class="meta-val">
          {{if .Receipt.Axoloom.Connected}}
            <span class="lang-en">Active ({{.Receipt.Axoloom.IndexedEndpoints}} routes, {{.Receipt.Axoloom.IndexedSQLTables}} SQL tables)</span>
            <span class="lang-id">Aktif ({{.Receipt.Axoloom.IndexedEndpoints}} route, {{.Receipt.Axoloom.IndexedSQLTables}} tabel SQL)</span>
          {{else}}
            <span class="lang-en">Standalone mode</span>
            <span class="lang-id">Mode mandiri</span>
          {{end}}
        </p>
      </div>
    </section>

    <section class="section">
      <h2><span class="lang-en">1. Business, Money &amp; Privacy Risks</span><span class="lang-id">1. Pemeriksaan Fitur Palsu, Kasir Uang &amp; Kunci Data</span></h2>
      <p class="section-desc">
        <span class="lang-en">Any issue here blocks launch because it can lose orders, miscalculate money, or expose customer records.</span>
        <span class="lang-id">Temuan di bagian ini wajib dibereskan sebelum dipakai pelanggan karena menyangkut data hilang, salah hitung uang, atau kebocoran privasi.</span>
      </p>
      {{if .Receipt.FatalRisks}}
        <ul class="item-list">
          {{range .Receipt.FatalRisks}}
            <li class="item-card fatal">
              <p class="item-title"><span class="lang-en">{{.TitleEN}}</span><span class="lang-id">{{.TitleID}}</span></p>
              <p class="item-impact"><span class="lang-en">{{.BusinessImpactEN}}</span><span class="lang-id">{{.BusinessImpactID}}</span></p>
              {{if .File}}<p class="item-meta">{{.File}}{{if .Line}}:{{.Line}}{{end}}</p>{{end}}
            </li>
          {{end}}
        </ul>
      {{else}}
        <p class="empty-ok">
          <span class="lang-en">Zero fatal placebo, money-rounding, or open-door risks detected.</span>
          <span class="lang-id">Bersih: tidak ditemukan tombol pajangan palsu, salah tipe uang, atau pintu data terbuka.</span>
        </p>
      {{end}}
    </section>

    <section class="section">
      <h2><span class="lang-en">2. Live Action &amp; Database Proof</span><span class="lang-id">2. Bukti Uji Jalan Nyata &amp; Simpan Database</span></h2>
      <p class="section-desc">
        <span class="lang-en">Out-of-band HTTP and database row measurements run directly by AxoProof (not self-reported by AI).</span>
        <span class="lang-id">Pengujian kirim data dan hitung baris database langsung oleh mesin AxoProof (bukan klaim sepihak AI).</span>
      </p>
      {{if .Receipt.RuntimeProbes}}
        <ul class="item-list">
          {{range .Receipt.RuntimeProbes}}
            <li class="item-card {{if and .Executed .Passed}}pass{{else if .Executed}}fatal{{else}}polish{{end}}">
              <p class="item-title">{{.Name}}</p>
              <p class="item-impact"><span class="lang-en">{{.SummaryEN}}</span><span class="lang-id">{{.SummaryID}}</span></p>
              <p class="item-meta">{{.RawEvidence}}</p>
            </li>
          {{end}}
        </ul>
      {{else}}
        <p class="item-card polish">
          <span class="lang-en">No live server URL was configured in .axoproof/claims.json yet. Static code checks ran, while live click/database persistence remains unproven.</span>
          <span class="lang-id">URL server lokal belum diisi pada .axoproof/claims.json. Kode sudah diperiksa statis, namun simpan database secara live belum diuji.</span>
        </p>
      {{end}}
    </section>

    <section class="section">
      <h2><span class="lang-en">3. Customer-Facing Copy &amp; Screen Readiness</span><span class="lang-id">3. Kesiapan Layar &amp; Bahasa Manusia untuk Pelanggan</span></h2>
      <p class="section-desc">
        <span class="lang-en">Separate non-fatal track catching fake AI sample contacts, dead '#' links, robotic brochure phrases, and tiny mobile buttons.</span>
        <span class="lang-id">Jalur terpisah untuk menangkap kontak contoh palsu, link '#' buntu, kalimat robotik klise AI, atau tombol HP kekecilan.</span>
      </p>
      {{if .Receipt.PolishWarnings}}
        <ul class="item-list">
          {{range .Receipt.PolishWarnings}}
            <li class="item-card polish">
              <p class="item-title"><span class="lang-en">{{.TitleEN}}</span><span class="lang-id">{{.TitleID}}</span></p>
              <p class="item-impact"><span class="lang-en">{{.BusinessImpactEN}}</span><span class="lang-id">{{.BusinessImpactID}}</span></p>
              {{if .File}}<p class="item-meta">{{.File}}{{if .Line}}:{{.Line}}{{end}}</p>{{end}}
            </li>
          {{end}}
        </ul>
      {{else}}
        <p class="empty-ok">
          <span class="lang-en">Clean customer screen: no fake sample contacts, dead '#' links, or robotic brochure cliches spotted.</span>
          <span class="lang-id">Layar bersih: bebas kontak contoh palsu, link buntu '#', maupun kalimat brosur robotik.</span>
        </p>
      {{end}}
    </section>

    <section class="section">
      <h2><span class="lang-en">4. Owner 2-Minute Manual Check (Honest Blindspots)</span><span class="lang-id">4. Daftar Cek Mandiri 2 Menit oleh Owner (Batas Jujur)</span></h2>
      <ul class="item-list">
        {{range .Receipt.Blindspots}}
          <li class="item-card">
            <p class="item-title"><span class="lang-en">{{.TitleEN}}</span><span class="lang-id">{{.TitleID}}</span></p>
            <p class="item-impact"><span class="lang-en">{{.WhyEN}}</span><span class="lang-id">{{.WhyID}}</span></p>
          </li>
        {{end}}
      </ul>
    </section>

    <details class="repair-drawer">
      <summary>
        <span class="lang-en">Machine Repair Instructions for AI Coding Agent ({{len .Receipt.FatalRisks}} fatal, {{len .Receipt.PolishWarnings}} polish)</span>
        <span class="lang-id">Instruksi Perbaikan Teknis untuk AI Agent ({{len .Receipt.FatalRisks}} fatal, {{len .Receipt.PolishWarnings}} polesan)</span>
      </summary>
      <pre class="repair-code">{{if or .Receipt.FatalRisks .Receipt.PolishWarnings}}{{range .Receipt.FatalRisks}}[FATAL / {{.ID}}] {{if .File}}{{.File}}:{{.Line}} - {{end}}{{.RepairInstruction}}
{{end}}{{range .Receipt.PolishWarnings}}[POLISH / {{.ID}}] {{if .File}}{{.File}}:{{.Line}} - {{end}}{{.RepairInstruction}}
{{end}}{{else}}All automated checks passed. No repair actions queued.{{end}}</pre>
    </details>

    <footer class="seal-footer">
      <span>Git: {{.Receipt.GitCommit}} ({{.Receipt.GitDirtyFiles}} uncommitted)</span>
      <span>SHA-256 Seal: <code>{{.Receipt.SealSHA256}}</code></span>
    </footer>
  </main>

  <script>
    (function () {
      var html = document.documentElement;
      var btns = document.querySelectorAll("[data-set-lang]");
      for (var i = 0; i < btns.length; i++) {
        btns[i].addEventListener("click", function () {
          var target = this.getAttribute("data-set-lang");
          html.setAttribute("data-lang", target);
          html.setAttribute("lang", target);
          for (var j = 0; j < btns.length; j++) {
            btns[j].setAttribute("aria-pressed", btns[j].getAttribute("data-set-lang") === target ? "true" : "false");
          }
        });
      }
    })();
  </script>
</body>
</html>
`
