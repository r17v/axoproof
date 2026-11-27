package scanners

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/r17v/axoproof/internal/model"
)

// ScanSummary contains all static findings plus the count of inspected source files.
type ScanSummary struct {
	FilesScanned   int
	FatalRisks     []model.Finding
	PolishWarnings []model.Finding
	DiscoveredURLs []string
}

var (
	// Track 1: Anti-Placebo patterns (dead handlers, fake hardcoded KPI currency, mock arrays in prod, skipped tests)
	reEmptyClick      = regexp.MustCompile(`on(Click|Submit)\s*=\s*\{\s*\([^)]*\)\s*=>\s*\{\s*(e\.preventDefault\(\)\s*;?\s*)?\}\s*\}`)
	reFakeDelaySave   = regexp.MustCompile(`setTimeout\s*\(\s*\([^)]*\)\s*=>\s*\{[^}]*(saved|success|berhasil|paid|terkirim)`)
	reHardcodedMetric = regexp.MustCompile(`(?i)(total\s+(revenue|sales|income|omzet|pendapatan)|today['’]?s\s+(revenue|sales)|pendapatan\s+hari\s+ini)[^<\n]{0,80}(Rp\s?[0-9][0-9.,]{4,}|\$[0-9][0-9,.]{3,})`)
	reMockProdIdent   = regexp.MustCompile(`\b(mockOrders|mockUsers|mockPayments|fakeTransactions|DUMMY_ORDERS|FAKE_REVENUE)\b`)
	reSkippedTest     = regexp.MustCompile(`\b(it|test|describe)\.skip\s*\(|^\s*t\.Skip\s*\(`)

	// Track 2: Cash-Register & Money Integrity patterns
	reSQLFloatMoney = regexp.MustCompile(`(?i)\b(price|unit_price|amount|total|subtotal|balance|tax|discount|fee|harga|tagihan|saldo)\b[^,\n;]{0,40}\b(FLOAT|REAL|DOUBLE\s+PRECISION|float64|float32)\b`)
	reGoFloatMoney  = regexp.MustCompile(`(?i)\b(price|unitPrice|amount|total|subtotal|balance|tax|discount|fee|harga|saldo)[A-Za-z0-9_]*\s+float(32|64)\b`)
	reJSFloatMoney  = regexp.MustCompile(`(?i)parseFloat\s*\([^)]*(price|amount|total|subtotal|discount|tax|harga|saldo)`)
	rePyFloatMoney  = regexp.MustCompile(`(?i)\b(price|amount|total|subtotal|discount|tax|harga|saldo)\s*=\s*float\s*\(`)
	reClientPrice   = regexp.MustCompile(`(?i)(req\.body|request\.json|body|payload|input)\s*(\.|\[['"])\s*(price|unit_price|total_amount|final_price|harga)\b`)

	// Track 3: Door-Locks & Customer Privacy patterns
	rePublicEnvSecret = regexp.MustCompile(`(NEXT_PUBLIC_|VITE_|REACT_APP_|PUBLIC_)[A-Z0-9_]*(SECRET|SERVICE_ROLE|PRIVATE_KEY|DB_PASSWORD|ADMIN_KEY)`)
	reLiveKeyLiteral  = regexp.MustCompile(`\b(sk_live_[A-Za-z0-9]{16,}|rk_live_[A-Za-z0-9]{16,}|ghp_[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16})\b`)
	reRawSQLConcat    = regexp.MustCompile(`(?i)(fmt\.Sprintf\s*\(\s*"\s*(SELECT|INSERT|UPDATE|DELETE)\b[^"]*%[sv]|` + "`" + `\s*(SELECT|INSERT|UPDATE|DELETE)\b[^` + "`" + `]*\$\{|f["']\s*(SELECT|INSERT|UPDATE|DELETE)\b[^"']*\{)`)
	reRouteDecl       = regexp.MustCompile(`(?i)(HandleFunc|Get|Post|Put|Patch|Delete|route)\s*\(\s*["'` + "`" + `](/api/[a-zA-Z0-9_/\-:]+)["'` + "`" + `]`)
	reSensitiveRoute  = regexp.MustCompile(`(?i)^/api/(admin|customers|users|orders|invoices|patients|payments|wallets|transactions|ledger)\b`)
	reAuthEvidence    = regexp.MustCompile(`(?i)\b(auth|session|jwt|bearer|token|cookie|currentUser|user_id|requireAuth|withAuth|middleware|Unauthorized|StatusUnauthorized|401|403)\b`)

	// Track 4: Customer-Ready & Human Copy Honesty (Non-fatal separate polish track)
	reFakeContact     = regexp.MustCompile(`(?i)(info@example\.com|support@example\.com|test@example\.com|lorem\s+ipsum|0812-?3456-?7890|\+1\s*\(?555\)?[\s-]*01[0-9]{2})`)
	reFakeSocialProof = regexp.MustCompile(`(?i)(trusted\s+by\s+(10,000|50,000|100,000)\+|dipercaya\s+oleh\s+(10\.000|50\.000|100\.000)\+)`)
	reDeadAnchor      = regexp.MustCompile(`href\s*=\s*["']#["']`)
	reRawStackLeak    = regexp.MustCompile(`\b(err\.stack|error\.stack|debug\.Stack\(\))\b`)
	reRoboticCopy     = regexp.MustCompile(`(?i)\b(revolutionize\s+your|unleash\s+the\s+power|seamlessly\s+integrated|in\s+today['’]?s\s+fast-paced\s+digital|solusi\s+komprehensif\s+revolusioner)\b`)
	reTinyTouchTarget = regexp.MustCompile(`(<button[^>]*class(Name)?=["'][^"']*\b(h-5|h-6|w-5|w-6)\b|<button[^>]*style=["'][^"']*height:\s*(1[6-9]|2[0-9]|3[0-2])px)`)
)

var ignoredDirs = map[string]bool{
	".git":         true,
	".axoproof":    true,
	".axoloom":     true,
	"node_modules": true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
}

var supportedExts = map[string]bool{
	".go":     true,
	".ts":     true,
	".tsx":    true,
	".js":     true,
	".jsx":    true,
	".py":     true,
	".sql":    true,
	".prisma": true,
	".html":   true,
	".env":    true,
}

func isTestFile(rel string) bool {
	lower := strings.ToLower(rel)
	return strings.HasSuffix(lower, "_test.go") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.HasPrefix(filepath.Base(lower), "test_") ||
		strings.Contains(lower, "/__tests__/") ||
		strings.Contains(lower, "\\__tests__\\")
}

// ScanWorkspace runs deterministic static inspection across supported source files.
func ScanWorkspace(workDir string) (ScanSummary, error) {
	var out ScanSummary
	seenRoutes := map[string]bool{}

	err := filepath.WalkDir(workDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != workDir && ignoredDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		if !supportedExts[ext] && !strings.HasPrefix(name, ".env") {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > 2*1024*1024 {
			return nil
		}

		rel, relErr := filepath.Rel(workDir, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		// Skip self-scanning AxoProof's own scanner definitions when running inside the axoproof repo itself.
		if rel == "internal/scanners/scanner.go" || rel == "internal/scanners/scanner_test.go" {
			return nil
		}

		out.FilesScanned++
		testFile := isTestFile(rel)

		rawBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fullText := string(rawBytes)
		hasFileAuthCheck := reAuthEvidence.MatchString(fullText)

		sc := bufio.NewScanner(strings.NewReader(fullText))
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "--") {
				continue
			}

			// Discover API routes for cross-checking
			if m := reRouteDecl.FindStringSubmatch(line); len(m) == 3 {
				routePath := m[2]
				if !seenRoutes[routePath] {
					seenRoutes[routePath] = true
					out.DiscoveredURLs = append(out.DiscoveredURLs, routePath)
				}
				if !testFile && reSensitiveRoute.MatchString(routePath) && !hasFileAuthCheck {
					out.FatalRisks = append(out.FatalRisks, model.Finding{
						ID:               "DOOR_OPEN_SENSITIVE_ROUTE",
						Track:            model.TrackFatalBusinessRisk,
						Module:           model.ModuleDoorLocks,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "Customer or admin data endpoint has no login lock",
						TitleID:          "Pintu data pelanggan atau admin terbuka tanpa kunci login",
						BusinessImpactEN: "Anyone with the link can read or modify sensitive records (" + routePath + ") without logging in.",
						BusinessImpactID: "Orang luar yang tahu alamat link (" + routePath + ") bisa mengintip atau mengubah data pelanggan tanpa perlu login.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Enforce session/JWT authentication middleware and verify resource ownership (user_id/tenant_id) on " + routePath + " in " + rel + ".",
					})
				}
			}

			// Skipped tests check (applies specifically to test files)
			if testFile {
				if reSkippedTest.MatchString(line) {
					out.FatalRisks = append(out.FatalRisks, model.Finding{
						ID:               "PLACEBO_SKIPPED_TEST",
						Track:            model.TrackFatalBusinessRisk,
						Module:           model.ModuleNoPlacebo,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "Automated test was silenced instead of fixed",
						TitleID:          "Pengujian otomatis sengaja dimatikan (skip) oleh AI",
						BusinessImpactEN: "A failing check was hidden with skip, meaning broken behavior can slip through undetected.",
						BusinessImpactID: "Pengujian yang gagal disembunyikan dengan fitur skip sehingga bug asli tidak ketahuan.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Remove .skip / t.Skip in " + rel + " and make the underlying test pass against real logic.",
					})
				}
				continue
			}

			// --- Production-only checks below ---

			// 1. No-Placebo checks
			if reEmptyClick.MatchString(line) {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "PLACEBO_DEAD_BUTTON",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleNoPlacebo,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Display-only button or form does nothing when clicked",
					TitleID:          "Tombol atau formulir pajangan (diklik tidak melakukan apa-apa)",
					BusinessImpactEN: "Customers can click this button on screen, but no action or data save is wired behind it.",
					BusinessImpactID: "Pelanggan bisa memencet tombol ini di layar, tetapi tidak ada proses penyimpanan apa pun di belakangnya.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Wire the onClick/onSubmit handler in " + rel + " to a real API mutation or state action and verify persistence.",
				})
			}
			if reFakeDelaySave.MatchString(line) {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "PLACEBO_FAKE_SAVE_TIMER",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleNoPlacebo,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Simulated loading timer pretending to save data",
					TitleID:          " Efek loading pura-pura menyimpan data (bohongan)",
					BusinessImpactEN: "The interface waits briefly and shows a success message without saving customer data.",
					BusinessImpactID: "Layar hanya menunggu sebentar lalu menampilkan pesan berhasil, padahal data tidak disimpan.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Replace the fake setTimeout success toast in " + rel + " with a real backend API call and database write.",
				})
			}
			if reHardcodedMetric.MatchString(line) || reMockProdIdent.MatchString(line) {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "PLACEBO_HARDCODED_FIGURES",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleNoPlacebo,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Dashboard or route displays hardcoded sample revenue/orders",
					TitleID:          "Angka omzet atau data pesanan masih teks karangan mati (hardcoded)",
					BusinessImpactEN: "Numbers shown to you on screen are static sample figures, not real live business transactions.",
					BusinessImpactID: "Angka yang tampil di layar Anda adalah angka contoh buatan AI, bukan hasil hitungan transaksi asli dari database.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Query real aggregated rows from the database instead of returning hardcoded figures or mock arrays in " + rel + ".",
				})
			}

			// 2. Cash-Register & Money checks
			if reSQLFloatMoney.MatchString(line) || reGoFloatMoney.MatchString(line) || reJSFloatMoney.MatchString(line) || rePyFloatMoney.MatchString(line) {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "MONEY_FLOAT_ROUNDING",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleCashRegister,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Money calculation uses imprecise decimal floating-point numbers",
					TitleID:          "Hitungan uang memakai tipe angka desimal (float) yang rawan selisih",
					BusinessImpactEN: "Invoices, discounts, taxes, or balances can silently lose or add cents/Rupiah on repeated transactions.",
					BusinessImpactID: "Tagihan, diskon, pajak, atau saldo bisa meleset beberapa perak/rupiah saat transaksi dijumlahkan.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Store and compute all monetary values as integer minor units (int64 cents/Rupiah) or exact SQL NUMERIC/DECIMAL in " + rel + ".",
				})
			}
			if reClientPrice.MatchString(line) && (strings.Contains(strings.ToLower(rel), "order") || strings.Contains(strings.ToLower(rel), "checkout") || strings.Contains(strings.ToLower(rel), "pay") || strings.Contains(strings.ToLower(rel), "cart") || ext == ".go" || ext == ".ts" || ext == ".py") {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "MONEY_CLIENT_PRICE_TRUST",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleCashRegister,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Checkout trusts item price sent directly from the buyer's browser",
					TitleID:          "Kasir mempercayai harga barang yang dikirim dari browser pembeli",
					BusinessImpactEN: "A buyer can edit browser request data and purchase expensive items for Rp 1 or $0.01.",
					BusinessImpactID: "Pembeli nakal bisa mengubah harga barang dari browser menjadi Rp 1 saat checkout.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Never trust client-submitted price fields in " + rel + "; look up authoritative unit prices from the server database by item SKU/ID.",
				})
			}

			// 3. Door-Locks & Secrets checks
			if rePublicEnvSecret.MatchString(line) || reLiveKeyLiteral.MatchString(line) {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "DOOR_EXPOSED_SECRET",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleDoorLocks,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Secret API or database key is exposed in code or browser config",
					TitleID:          "Kunci rahasia API atau database terpampang terbuka di kode",
					BusinessImpactEN: "Visitors can inspect your website bundle or repo, steal this key, and drain your account or database.",
					BusinessImpactID: "Orang luar bisa mengambil kunci rahasia ini dari website untuk menguras kuota API atau isi database Anda.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Move the secret key out of " + rel + " into a server-only environment variable (never NEXT_PUBLIC_ or VITE_) and rotate the exposed credential.",
				})
			}
			if reRawSQLConcat.MatchString(line) {
				out.FatalRisks = append(out.FatalRisks, model.Finding{
					ID:               "DOOR_RAW_SQL_INJECTION",
					Track:            model.TrackFatalBusinessRisk,
					Module:           model.ModuleDoorLocks,
					File:             rel,
					Line:             lineNo,
					TitleEN:          "Database query builds SQL directly from raw text input",
					TitleID:          "Pencarian database menyambung teks mentah yang rawan dibobol",
					BusinessImpactEN: "An attacker can type crafted input into a form to dump or wipe your database tables.",
					BusinessImpactID: "Penyerang bisa memasukkan teks khusus pada form untuk menyedot atau menghapus isi tabel database.",
					EvidenceSnippet:  truncateSnippet(trimmed),
					RepairInstruction: "Replace string-concatenated SQL in " + rel + " with parameterized placeholders ($1, ?) and bound arguments.",
				})
			}

			// 4. Customer-Ready & Human Copy Honesty (Separate Polish Track!)
			if ext == ".tsx" || ext == ".jsx" || ext == ".html" || ext == ".ts" || ext == ".js" || ext == ".go" {
				if reFakeContact.MatchString(line) || reFakeSocialProof.MatchString(line) {
					out.PolishWarnings = append(out.PolishWarnings, model.Finding{
						ID:               "POLISH_FAKE_PLACEHOLDER_TRUST",
						Track:            model.TrackCustomerPolish,
						Module:           model.ModuleCustomerReady,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "AI sample contact details or unverified customer-count claim left on screen",
						TitleID:          "Kontak contoh (example.com / nomor palsu) atau klaim jumlah pelanggan karangan AI masih tertinggal",
						BusinessImpactEN: "Real visitors will immediately spot fake contact info or made-up '10,000+ customers' claims and lose trust in your brand.",
						BusinessImpactID: "Calon pembeli asli akan langsung sadar ada email/nomor telepon contoh atau klaim angka karangan AI sehingga ragu bertransaksi.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Replace placeholder contacts/claims in " + rel + " with verified business details from the owner or Axoloom vault.",
					})
				}
				if reDeadAnchor.MatchString(line) {
					out.PolishWarnings = append(out.PolishWarnings, model.Finding{
						ID:               "POLISH_DEAD_LINK",
						Track:            model.TrackCustomerPolish,
						Module:           model.ModuleCustomerReady,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "Navigation or footer link points nowhere (href='#')",
						TitleID:          "Tautan menu atau tombol mengarah ke jalan buntu (href='#')",
						BusinessImpactEN: "Clicking this link leaves customers stuck on the same page, making the site feel unfinished.",
						BusinessImpactID: "Saat diklik pelanggan, tautan ini buntu dan membuat website terasa belum selesai dikerjakan.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Point the anchor tag in " + rel + " to a real destination URL/route, or change it into a <button type=\"button\"> with a real handler.",
					})
				}
				if reRawStackLeak.MatchString(line) {
					out.PolishWarnings = append(out.PolishWarnings, model.Finding{
						ID:               "POLISH_RAW_ERROR_EXPOSED",
						Track:            model.TrackCustomerPolish,
						Module:           model.ModuleCustomerReady,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "Raw technical error trace can leak onto customer screens",
						TitleID:          "Pesan error koding mentah bisa bocor ke layar pelanggan",
						BusinessImpactEN: "When something fails, customers see scary internal code traces instead of a helpful human message.",
						BusinessImpactID: "Saat terjadi kendala, pelanggan melihat teks koding internal yang membingungkan, bukan panduan bahasa manusia.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Log internal stack traces on the server only and return a clear, polite human error message in " + rel + ".",
					})
				}
				if reRoboticCopy.MatchString(line) {
					out.PolishWarnings = append(out.PolishWarnings, model.Finding{
						ID:               "POLISH_ROBOTIC_AI_PROSE",
						Track:            model.TrackCustomerPolish,
						Module:           model.ModuleCustomerReady,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "Interface text uses generic AI brochure cliches",
						TitleID:          "Kalimat tampilan memakai bahasa brosur robotik khas AI",
						BusinessImpactEN: "Generic AI hype phrases make your business sound automated rather than written by real people.",
						BusinessImpactID: "Kalimat klise khas robot mengurangi kesan tepercaya dan manusiawi dari bisnis Anda.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Rewrite the copy in " + rel + " using plain, concrete words stating exactly what the customer gets.",
					})
				}
				if reTinyTouchTarget.MatchString(line) {
					out.PolishWarnings = append(out.PolishWarnings, model.Finding{
						ID:               "POLISH_TINY_MOBILE_BUTTON",
						Track:            model.TrackCustomerPolish,
						Module:           model.ModuleCustomerReady,
						File:             rel,
						Line:             lineNo,
						TitleEN:          "Button is too small to tap comfortably on a phone screen (<44px)",
						TitleID:          "Ukuran tombol terlalu kecil untuk dipencet jempol di layar HP (<44px)",
						BusinessImpactEN: "Mobile customers may mis-tap or struggle to press this control.",
						BusinessImpactID: "Pelanggan yang membuka dari HP akan kesulitan memencet tombol ini dengan jempol.",
						EvidenceSnippet:  truncateSnippet(trimmed),
						RepairInstruction: "Increase interactive button dimensions/padding in " + rel + " to at least 44x44px.",
					})
				}
			}
		}
		return nil
	})

	return out, err
}

func truncateSnippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 140 {
		return s[:137] + "..."
	}
	return s
}
