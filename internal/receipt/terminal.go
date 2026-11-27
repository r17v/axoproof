package receipt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/r17v/axoproof/internal/model"
)

type ansiPalette struct {
	bold  string
	dim   string
	green string
	amber string
	red   string
	reset string
}

func detectANSI(enableColor bool) ansiPalette {
	if !enableColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return ansiPalette{}
	}
	return ansiPalette{
		bold:  "\033[1m",
		dim:   "\033[2m",
		green: "\033[32m",
		amber: "\033[33m",
		red:   "\033[31m",
		reset: "\033[0m",
	}
}

// IsStdoutTTY checks whether os.Stdout is an interactive character device.
func IsStdoutTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// RenderTerminal formats a 76-column terminal inspection ledger readable on macOS, Windows, Linux SSH, and Android Termux.
func RenderTerminal(r model.SealedReceipt, htmlPath string, lang string, color bool) string {
	isID := strings.ToLower(lang) == "id"
	c := detectANSI(color)
	hr := strings.Repeat("─", 74)
	thin := strings.Repeat("·", 74)

	var b strings.Builder
	b.WriteString(hr + "\n")

	statusBadge := "[PENDING]"
	statusColor := c.amber
	headline := r.HeadlineEN
	if isID {
		headline = r.HeadlineID
	}
	switch r.Verdict {
	case model.VerdictSafeToTest:
		if isID {
			statusBadge = "[AMAN DITES]"
		} else {
			statusBadge = "[VERIFIED]"
		}
		statusColor = c.green
	case model.VerdictHoldRelease:
		if isID {
			statusBadge = "[TAHAN RILIS]"
		} else {
			statusBadge = "[HOLD RELEASE]"
		}
		statusColor = c.red
	default:
		if isID {
			statusBadge = "[CEK MANUAL]"
		}
	}

	titleLabel := "AxoProof Inspection Ledger"
	if isID {
		titleLabel = "AxoProof Lembar Pemeriksaan CLI"
	}
	fmt.Fprintf(&b, "%s%s%s  (%s)\n", c.bold, titleLabel, c.reset, r.ProjectName)
	fmt.Fprintf(&b, "%s%s %s%s\n", statusColor+c.bold, statusBadge, headline, c.reset)
	b.WriteString(hr + "\n")

	axoStatus := "Standalone"
	if isID {
		axoStatus = "Mandiri"
	}
	if r.Axoloom.Connected {
		if isID {
			axoStatus = fmt.Sprintf("Aktif (%d route, %d tabel)", r.Axoloom.IndexedEndpoints, r.Axoloom.IndexedSQLTables)
		} else {
			axoStatus = fmt.Sprintf("Active (%d routes, %d tables)", r.Axoloom.IndexedEndpoints, r.Axoloom.IndexedSQLTables)
		}
	}

	if isID {
		fmt.Fprintf(&b, "Berkas diperiksa : %-6d | Masalah fatal : %d | Catatan layar : %d\n",
			r.FilesScanned, len(r.FatalRisks), len(r.PolishWarnings))
		fmt.Fprintf(&b, "Uji jalan live   : %-6d | Otak Axoloom  : %s\n",
			len(r.RuntimeProbes), axoStatus)
	} else {
		fmt.Fprintf(&b, "Files inspected  : %-6d | Blocking risks : %d | Screen notes : %d\n",
			r.FilesScanned, len(r.FatalRisks), len(r.PolishWarnings))
		fmt.Fprintf(&b, "Live probes      : %-6d | Axoloom link   : %s\n",
			len(r.RuntimeProbes), axoStatus)
	}

	// Section 01: Fatal Business, Money & Privacy
	b.WriteString(thin + "\n")
	if isID {
		fmt.Fprintf(&b, "%s01. Kasir, Kunci Data & Fitur Asli%s\n", c.bold, c.reset)
	} else {
		fmt.Fprintf(&b, "%s01. Money, Privacy & Real Wiring%s\n", c.bold, c.reset)
	}
	if len(r.FatalRisks) == 0 {
		if isID {
			fmt.Fprintf(&b, "  %s[LULUS]%s Bersih dari tombol pajangan, salah tipe uang, dan pintu terbuka.\n", c.green, c.reset)
		} else {
			fmt.Fprintf(&b, "  %s[PASS]%s  No fake buttons, float currency bugs, or unlocked routes.\n", c.green, c.reset)
		}
	} else {
		for i, f := range r.FatalRisks {
			t := f.TitleEN
			imp := f.BusinessImpactEN
			if isID {
				t = f.TitleID
				imp = f.BusinessImpactID
			}
			loc := ""
			if f.File != "" {
				loc = fmt.Sprintf(" [%s:%d]", f.File, f.Line)
			}
			fmt.Fprintf(&b, "  %s%d. %s%s%s\n", c.red+c.bold, i+1, t, c.reset, c.dim+loc+c.reset)
			fmt.Fprintf(&b, "     %s\n", imp)
		}
	}

	// Section 02: Live Runtime Measurements
	b.WriteString(thin + "\n")
	if isID {
		fmt.Fprintf(&b, "%s02. Pengukuran Server & Database Live%s\n", c.bold, c.reset)
	} else {
		fmt.Fprintf(&b, "%s02. Live Server & Database Ledger%s\n", c.bold, c.reset)
	}
	if len(r.RuntimeProbes) == 0 {
		if isID {
			fmt.Fprintf(&b, "  %s[TUNDA]%s Server lokal belum dikonfigurasi pada .axoproof/claims.json.\n", c.amber, c.reset)
		} else {
			fmt.Fprintf(&b, "  %s[SKIP]%s  No live BaseURL configured in .axoproof/claims.json yet.\n", c.amber, c.reset)
		}
	} else {
		for _, p := range r.RuntimeProbes {
			tag := "[SKIP]"
			col := c.amber
			if p.Executed && p.Passed {
				if isID {
					tag = "[BUKTI]"
				} else {
					tag = "[PROVEN]"
				}
				col = c.green
			} else if p.Executed && !p.Passed {
				if isID {
					tag = "[GAGAL]"
				} else {
					tag = "[FAILED]"
				}
				col = c.red
			}
			fmt.Fprintf(&b, "  %s%-8s%s %-36s %s%s%s\n", col+c.bold, tag, c.reset, clipText(p.Name, 36), c.dim, p.RawEvidence, c.reset)
		}
	}

	// Section 03: Customer Screen & Human Copy (only expand rows when warnings exist)
	if len(r.PolishWarnings) > 0 {
		b.WriteString(thin + "\n")
		if isID {
			fmt.Fprintf(&b, "%s03. Catatan Layar & Bahasa Manusia (%d)%s\n", c.bold, len(r.PolishWarnings), c.reset)
		} else {
			fmt.Fprintf(&b, "%s03. Customer Screen & Human Copy (%d)%s\n", c.bold, len(r.PolishWarnings), c.reset)
		}
		for i, w := range r.PolishWarnings {
			t := w.TitleEN
			if isID {
				t = w.TitleID
			}
			loc := ""
			if w.File != "" {
				loc = fmt.Sprintf(" [%s:%d]", w.File, w.Line)
			}
			fmt.Fprintf(&b, "  %s%d. %s%s%s\n", c.amber, i+1, t, c.reset, c.dim+loc+c.reset)
		}
	}

	// Section 04: 2-Minute Owner Check
	if len(r.Blindspots) > 0 {
		b.WriteString(thin + "\n")
		if isID {
			fmt.Fprintf(&b, "%s04. Cek Mata Owner 2 Menit%s\n", c.bold, c.reset)
		} else {
			fmt.Fprintf(&b, "%s04. Owner 2-Minute Manual Check%s\n", c.bold, c.reset)
		}
		for _, bs := range r.Blindspots {
			t := bs.TitleEN
			if isID {
				t = bs.TitleID
			}
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}

	b.WriteString(hr + "\n")
	shortSeal := r.SealSHA256
	if len(shortSeal) > 16 {
		shortSeal = shortSeal[:16]
	}
	if isID {
		fmt.Fprintf(&b, "Dokumen Lengkap : %s (tambah --open bila ingin buka di browser)\n", htmlPath)
		fmt.Fprintf(&b, "Segel SHA-256   : %s... | Git: %s\n", shortSeal, r.GitCommit)
	} else {
		fmt.Fprintf(&b, "Full Dossier    : %s (pass --open to view in browser)\n", htmlPath)
		fmt.Fprintf(&b, "SHA-256 Seal    : %s... | Git: %s\n", shortSeal, r.GitCommit)
	}
	b.WriteString(hr + "\n")
	return b.String()
}

// RenderMarkdown produces a compact GitHub-Flavored Markdown summary table for AI Harness chat replies.
func RenderMarkdown(r model.SealedReceipt, htmlPath string, lang string) string {
	isID := strings.ToLower(lang) == "id"
	absHTML, err := filepath.Abs(htmlPath)
	if err != nil {
		absHTML = htmlPath
	}
	fileURI := "file://" + filepath.ToSlash(absHTML)
	shortSeal := r.SealSHA256
	if len(shortSeal) > 16 {
		shortSeal = shortSeal[:16]
	}

	passedProbes := 0
	for _, p := range r.RuntimeProbes {
		if p.Executed && p.Passed {
			passedProbes++
		}
	}

	var b strings.Builder
	if isID {
		fmt.Fprintf(&b, "### Hasil Pemeriksaan AxoProof: %s\n\n", r.HeadlineID)
		b.WriteString("| Pemeriksaan | Hasil | Rincian |\n")
		b.WriteString("| :--- | :--- | :--- |\n")
		fmt.Fprintf(&b, "| **Kasir, Privasi & Fitur Asli** | `%d temuan` | %d berkas diperiksa |\n", len(r.FatalRisks), r.FilesScanned)
		fmt.Fprintf(&b, "| **Uji Database & Pintu Live** | `%d / %d lulus` | Segel SHA-256 `%s` |\n", passedProbes, len(r.RuntimeProbes), shortSeal)
		fmt.Fprintf(&b, "| **Kerapian Layar & Bahasa** | `%d catatan` | [Buka Lembar Pemeriksaan Lengkap](%s) |\n", len(r.PolishWarnings), fileURI)
	} else {
		fmt.Fprintf(&b, "### AxoProof Verification: %s\n\n", r.HeadlineEN)
		b.WriteString("| Check | Result | Detail |\n")
		b.WriteString("| :--- | :--- | :--- |\n")
		fmt.Fprintf(&b, "| **Money, Privacy & Real Wiring** | `%d blocking` | %d source files inspected |\n", len(r.FatalRisks), r.FilesScanned)
		fmt.Fprintf(&b, "| **Live Server & DB Probes** | `%d / %d proven` | SHA-256 seal `%s` |\n", passedProbes, len(r.RuntimeProbes), shortSeal)
		fmt.Fprintf(&b, "| **Customer Screen & Copy** | `%d notes` | [Open Full Inspection Dossier](%s) |\n", len(r.PolishWarnings), fileURI)
	}
	return b.String()
}

func clipText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
