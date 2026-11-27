package probes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/r17v/axoproof/internal/model"
)

var (
	reCrashOverlay = regexp.MustCompile(`(?i)(Application error:\s*a client-side exception|Unhandled Runtime Error|Cannot read properties of (undefined|null)|Hydration failed because the initial UI|Internal Server Error)`)
	reNaNCurrency  = regexp.MustCompile(`(\bRp\s*(NaN|undefined|null)\b|\$(NaN|undefined|null)\b|\bNaN%\b)`)
	reHTMLTags     = regexp.MustCompile(`(?s)<script\b[^>]*>.*?</script>|<style\b[^>]*>.*?</style>|<[^>]+>`)
)

func detectHeadlessBrowser() string {
	if custom := strings.TrimSpace(os.Getenv("AXOPROOF_BROWSER_BIN")); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom
		}
	}
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		}
	default:
		for _, bin := range []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"} {
			if p, err := exec.LookPath(bin); err == nil && p != "" {
				return p
			}
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func runHeadlessScreenProbes(base string, routes []string) ([]model.ProbeResult, []model.Finding) {
	if len(routes) == 0 {
		return nil, nil
	}
	browserBin := detectHeadlessBrowser()
	if browserBin == "" {
		return []model.ProbeResult{
			{
				Name:        "Headless browser screen render check",
				Kind:        "screen_dom",
				Executed:    false,
				Passed:      false,
				Target:      strings.Join(routes, ", "),
				SummaryEN:   "Skipped headless screen render check: no local Chrome/Edge/Chromium binary detected.",
				SummaryID:   "Pemeriksaan layar browser dilewati: Chrome/Edge/Chromium tidak ditemukan di perangkat ini.",
				RawEvidence: "browser binary not installed",
			},
		}, nil
	}

	var results []model.ProbeResult
	var findings []model.Finding

	for _, r := range routes {
		route := strings.TrimSpace(r)
		if route == "" {
			continue
		}
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		targetURL := base + route

		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		cmd := exec.CommandContext(ctx, browserBin,
			"--headless=new",
			"--disable-gpu",
			"--no-first-run",
			"--no-default-browser-check",
			"--window-size=375,812",
			"--virtual-time-budget=2000",
			"--dump-dom",
			targetURL,
		)
		domBytes, err := cmd.Output()
		cancel()

		if err != nil {
			results = append(results, model.ProbeResult{
				Name:        "Mobile 375px screen render (" + route + ")",
				Kind:        "screen_dom",
				Executed:    false,
				Passed:      false,
				Target:      targetURL,
				SummaryEN:   "Headless browser could not render " + route + " within timeout.",
				SummaryID:   "Browser headless gagal membuka " + route + " dalam batas waktu.",
				RawEvidence: err.Error(),
			})
			continue
		}

		dom := string(domBytes)
		visibleText := strings.TrimSpace(reHTMLTags.ReplaceAllString(dom, " "))
		visibleLen := len(strings.Fields(visibleText))

		if m := reCrashOverlay.FindString(dom); m != "" || visibleLen == 0 {
			reason := "blank empty screen (0 visible words)"
			if m != "" {
				reason = m
			}
			results = append(results, model.ProbeResult{
				Name:        "Mobile 375px screen render (" + route + ")",
				Kind:        "screen_dom",
				Executed:    true,
				Passed:      false,
				Target:      targetURL,
				SummaryEN:   fmt.Sprintf("Screen %s crashed or rendered completely blank (%s).", route, reason),
				SummaryID:   fmt.Sprintf("Halaman %s tampil putih kosong atau error JavaScript (%s).", route, reason),
				RawEvidence: "DOM crash/blank: " + reason,
			})
			findings = append(findings, model.Finding{
				ID:                "RUNTIME_SCREEN_CRASH_OR_BLANK",
				Track:             model.TrackFatalBusinessRisk,
				Module:            model.ModuleNoPlacebo,
				TitleEN:           fmt.Sprintf("Page %s renders a blank screen or client crash in browser", route),
				TitleID:           fmt.Sprintf("Halaman %s tampil putih kosong atau error saat dibuka di browser", route),
				BusinessImpactEN:  fmt.Sprintf("Opening %s in a real 375px phone browser resulted in %s.", route, reason),
				BusinessImpactID:  fmt.Sprintf("Saat halaman %s dibuka di browser ukuran HP 375px, layar kosong atau muncul error (%s).", route, reason),
				EvidenceSnippet:   reason,
				RepairInstruction: fmt.Sprintf("Inspect client hydration/JS runtime errors on route %s and ensure visible content renders cleanly.", route),
			})
			continue
		}

		if nanHit := reNaNCurrency.FindString(visibleText); nanHit != "" {
			results = append(results, model.ProbeResult{
				Name:        "Mobile 375px screen render (" + route + ")",
				Kind:        "screen_dom",
				Executed:    true,
				Passed:      false,
				Target:      targetURL,
				SummaryEN:   fmt.Sprintf("Screen %s displays broken number text (%s) to customers.", route, nanHit),
				SummaryID:   fmt.Sprintf("Halaman %s menampilkan angka rusak (%s) ke layar pelanggan.", route, nanHit),
				RawEvidence: "Rendered broken value: " + nanHit,
			})
			findings = append(findings, model.Finding{
				ID:                "RUNTIME_SCREEN_BROKEN_NUMBER",
				Track:             model.TrackFatalBusinessRisk,
				Module:            model.ModuleCashRegister,
				TitleEN:           fmt.Sprintf("Page %s displays broken NaN/undefined price or number on screen", route),
				TitleID:           fmt.Sprintf("Halaman %s menampilkan tulisan harga/angka rusak (%s) di layar", route, nanHit),
				BusinessImpactEN:  fmt.Sprintf("Customers opening %s see broken calculation output (%s) on screen.", route, nanHit),
				BusinessImpactID:  fmt.Sprintf("Pelanggan yang membuka halaman %s melihat angka error (%s) langsung di layar.", route, nanHit),
				EvidenceSnippet:   nanHit,
				RepairInstruction: fmt.Sprintf("Guard missing/null numeric fields and format currency safely on route %s.", route),
			})
			continue
		}

		results = append(results, model.ProbeResult{
			Name:        "Mobile 375px screen render (" + route + ")",
			Kind:        "screen_dom",
			Executed:    true,
			Passed:      true,
			Target:      targetURL,
			SummaryEN:   fmt.Sprintf("Rendered cleanly in real headless browser (%d visible words, zero JS crash or NaN).", visibleLen),
			SummaryID:   fmt.Sprintf("Tampil utuh di browser asli ukuran HP (%d kata terbaca, nol layar putih atau angka NaN).", visibleLen),
			RawEvidence: fmt.Sprintf("375x812 DOM ok (%d words)", visibleLen),
		})
	}

	return results, findings
}
