package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/r17v/axoproof/internal/axobridge"
	"github.com/r17v/axoproof/internal/claims"
	"github.com/r17v/axoproof/internal/model"
	"github.com/r17v/axoproof/internal/probes"
	"github.com/r17v/axoproof/internal/receipt"
	"github.com/r17v/axoproof/internal/scanners"
)

const version = "v0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	sub := os.Args[1]
	switch sub {
	case "version", "--version", "-v":
		fmt.Printf("axoproof %s\n", version)
	case "init":
		runInit(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "check-seal":
		runCheckSeal(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %q\n\n", sub)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`AxoProof - Independent Proof & Human-Ready Receipt Engine for AI Coding Agents

Usage:
  axoproof init       [--dir <path>]
  axoproof verify     [--dir <path>] [--claims <path>] [--scope all|diff] [--lang en|id] [--format table|markdown|json] [--sync-axoloom] [--open]
  axoproof check-seal [--dir <path>] [<receipt.json>]
  axoproof version`)
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dir := fs.String("dir", ".", "Workspace directory")
	_ = fs.Parse(args)

	outPath, err := claims.WriteTemplate(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "axoproof init failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Initialized starter claims manifest at %s\n", outPath)
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	dir := fs.String("dir", ".", "Workspace directory to verify")
	claimsPath := fs.String("claims", "", "Optional path to claims.json")
	scope := fs.String("scope", "all", "Inspection scope: all (entire workspace) or diff (git changed files + Axoloom callers)")
	lang := fs.String("lang", "en", "Default receipt display language (en or id)")
	format := fs.String("format", "table", "Output format: table (CLI/Termux), markdown (AI Harness chat), or json")
	syncAxo := fs.Bool("sync-axoloom", false, "Write sealed verification outcome back to Axoloom handoff memory")
	openBrowser := fs.Bool("open", false, "Explicitly open RECEIPT.html in desktop browser (off by default)")
	allowRemote := fs.Bool("allow-remote-probe", false, "Allow HTTP probes against non-loopback staging URLs")
	asJSON := fs.Bool("json", false, "Shorthand for --format json")
	_ = fs.Parse(args)

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve workspace: %v\n", err)
		os.Exit(1)
	}

	manifest, _, err := claims.Load(absDir, *claimsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "axoproof verify error: %v\n", err)
		os.Exit(1)
	}

	var allowedFiles map[string]bool
	if strings.EqualFold(strings.TrimSpace(*scope), "diff") {
		diffSet := scanners.ResolveGitDiffFiles(absDir)
		if len(diffSet) > 0 {
			allowedFiles = axobridge.ExpandBlastRadiusFiles(absDir, diffSet)
		}
	}

	scanRes, err := scanners.ScanWorkspaceFiltered(absDir, allowedFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan workspace error: %v\n", err)
		os.Exit(1)
	}

	axoCtx := axobridge.Inspect(absDir, manifest.ProtectedRoutes)
	probeOut := probes.RunAll(absDir, manifest, axoCtx.UndeclaredRoutesFound, *allowRemote)

	allFatal := append([]model.Finding{}, scanRes.FatalRisks...)
	allFatal = append(allFatal, probeOut.FatalFindings...)

	blindspots := buildBlindspots(manifest, probeOut)
	verdict, headEN, headID, subEN, subID := computeVerdict(allFatal, scanRes.PolishWarnings, probeOut)
	gitSHA, gitDirty := inspectGitState(absDir)

	rec := model.SealedReceipt{
		SchemaVersion:    "1.0",
		ProjectName:      manifest.ProjectName,
		WorkspaceSummary: manifest.Summary,
		GeneratedAtUTC:   time.Now().UTC().Format(time.RFC3339),
		GitCommit:        gitSHA,
		GitDirtyFiles:    gitDirty,
		Verdict:          verdict,
		HeadlineEN:       headEN,
		HeadlineID:       headID,
		SubheadlineEN:    subEN,
		SubheadlineID:    subID,
		FilesScanned:     scanRes.FilesScanned,
		FatalRisks:       allFatal,
		PolishWarnings:   scanRes.PolishWarnings,
		RuntimeProbes:    probeOut.Results,
		Blindspots:       blindspots,
		Axoloom:          axoCtx,
	}

	if *syncAxo {
		// Pre-compute seal once if syncing back, then re-seal cleanly in WriteArtifacts
		if tmpSeal, serr := receipt.ComputeSeal(rec); serr == nil {
			rec.SealSHA256 = tmpSeal
			rec.Axoloom.WritebackPath = axobridge.SyncBack(absDir, rec)
		}
	}

	sealedRec, _, htmlPath, err := receipt.WriteArtifacts(absDir, rec, strings.ToLower(*lang))
	if err != nil {
		fmt.Fprintf(os.Stderr, "write receipt artifacts: %v\n", err)
		os.Exit(1)
	}

	if *openBrowser {
		openInBrowser(htmlPath)
	}

	outFormat := strings.ToLower(strings.TrimSpace(*format))
	if *asJSON {
		outFormat = "json"
	}

	switch outFormat {
	case "json":
		enc, _ := json.MarshalIndent(sealedRec, "", "  ")
		fmt.Println(string(enc))
	case "markdown", "md":
		fmt.Print(receipt.RenderMarkdown(sealedRec, htmlPath, *lang))
	default:
		fmt.Print(receipt.RenderTerminal(sealedRec, htmlPath, *lang, receipt.IsStdoutTTY()))
	}

	if sealedRec.Verdict == model.VerdictHoldRelease {
		os.Exit(2)
	}
}

func runCheckSeal(args []string) {
	fs := flag.NewFlagSet("check-seal", flag.ExitOnError)
	dir := fs.String("dir", ".", "Workspace directory")
	_ = fs.Parse(args)

	target := filepath.Join(*dir, ".axoproof", "receipt.json")
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read receipt: %v\n", err)
		os.Exit(1)
	}
	var r model.SealedReceipt
	if err := json.Unmarshal(raw, &r); err != nil {
		fmt.Fprintf(os.Stderr, "parse receipt: %v\n", err)
		os.Exit(1)
	}
	if !receipt.VerifySeal(r) {
		fmt.Fprintf(os.Stderr, "TAMPERED OR INVALID SEAL on %s\n", target)
		os.Exit(2)
	}
	fmt.Printf("VALID SHA-256 SEAL (%s) for project %q [%s]\n", r.SealSHA256[:16], r.ProjectName, r.Verdict)
}

func computeVerdict(fatal []model.Finding, polish []model.Finding, probeOut probes.Outcome) (model.VerdictStatus, string, string, string, string) {
	if len(fatal) > 0 {
		return model.VerdictHoldRelease,
			"Hold release until blocking defects are fixed",
			"Tahan rilis sebelum masalah utama dibereskan",
			fmt.Sprintf("AxoProof found %d issue(s) that could lose orders, miscalculate money, or expose private records. Click the button below to copy repair instructions for your AI agent.", len(fatal)),
			fmt.Sprintf("AxoProof menemukan %d masalah yang bisa menghilangkan pesanan, melesetkan hitungan uang, atau membuka data pelanggan. Klik tombol di bawah untuk menyalin perintah perbaikan ke AI Agent.", len(fatal))
	}

	hasPersistenceProof := false
	allProbesPassed := probeOut.AnyExecuted && probeOut.AllExecuted
	for _, p := range probeOut.Results {
		if !p.Passed || !p.Executed {
			allProbesPassed = false
		}
		if p.Kind == "db_persistence" && p.Executed && p.Passed {
			hasPersistenceProof = true
		}
	}

	if allProbesPassed && hasPersistenceProof {
		if len(polish) > 0 {
			return model.VerdictSafeToTest,
				"Verified live, with screen polish notes",
				"Terbukti jalan nyata, dengan catatan kerapian layar",
				fmt.Sprintf("Database writes, pricing rules, and login locks passed live checks. %d customer-screen note(s) are listed in section 03.", len(polish)),
				fmt.Sprintf("Simpan database nyata, aturan harga, dan kunci login lulus uji langsung. Ada %d catatan kerapian layar di bagian 03.", len(polish))
		}
		return model.VerdictSafeToTest,
			"Verified live and ready for customer testing",
			"Terbukti jalan nyata dan siap diuji ke pelanggan",
			"Every inspected code path, login lock, and live database row count passed independent measurement.",
			"Pemeriksaan kode, kunci pintu login, dan pertambahan isi database telah terbukti langsung oleh mesin."
	}

	return model.VerdictLocalOnlyManualCheck,
		"Source code clean, live database check pending",
		"Kode bersih, menunggu uji simpan database langsung",
		"No static defects were found, while live server/database row counts have not been measured yet. Review section 04 before launch.",
		"Tidak ditemukan kode palsu atau celah statis, namun pertambahan baris database belum diukur secara live. Cek bagian 04 sebelum rilis."
}

func buildBlindspots(m claims.Manifest, probeOut probes.Outcome) []model.BlindspotItem {
	var list []model.BlindspotItem
	for _, ext := range m.ExternalDeps {
		list = append(list, model.BlindspotItem{
			TitleEN: ext.Service,
			TitleID: ext.Service,
			WhyEN:   ext.NoteEN,
			WhyID:   ext.NoteID,
		})
	}
	if !probeOut.AnyExecuted {
		list = append(list, model.BlindspotItem{
			TitleEN: "Live click-to-database submission",
			TitleID: "Tes klik simpan data secara langsung",
			WhyEN:   "Start the app locally and wire .axoproof/claims.json BaseURL so AxoProof can verify real database row changes automatically.",
			WhyID:   "Nyalakan server aplikasi dan isi BaseURL di .axoproof/claims.json agar AxoProof bisa menghitung pertambahan baris database otomatis.",
		})
	}
	list = append(list, model.BlindspotItem{
		TitleEN: "Real bank/QRIS/Stripe settlement & business offer accuracy",
		TitleID: "Uang masuk rekening asli (QRIS/Bank/Stripe) & kesesuaian harga bisnis",
		WhyEN:   "Automated local probes never move real bank funds; always confirm one live payment and verify your real product pricing before public ads.",
		WhyID:   "Pengujian lokal tidak memindahkan uang bank asli; selalu tes 1 pembayaran kecil nyata dan pastikan daftar harga sesuai katalog asli Anda.",
	})
	return list
}

func inspectGitState(workDir string) (string, int) {
	sha := "uncommitted-workspace"
	if out, err := exec.Command("git", "-C", workDir, "rev-parse", "--short", "HEAD").Output(); err == nil {
		s := strings.TrimSpace(string(out))
		if s != "" {
			sha = s
		}
	}
	dirty := 0
	if out, err := exec.Command("git", "-C", workDir, "status", "--porcelain").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if strings.TrimSpace(line) != "" {
				dirty++
			}
		}
	}
	return sha, dirty
}

func openInBrowser(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}
