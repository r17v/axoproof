package probes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/r17v/axoproof/internal/claims"
	"github.com/r17v/axoproof/internal/model"
	_ "modernc.org/sqlite"
)

var validSQLIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Outcome bundles probe logs plus any fatal defects proven live against the running server.
type Outcome struct {
	Results       []model.ProbeResult
	FatalFindings []model.Finding
	AllExecuted   bool
	AnyExecuted   bool
	AutoStarted   bool
}

// RunAll executes counterfactual HTTP, SQLite/Read-Back persistence, door-lock, tamper, and headless screen probes.
func RunAll(workDir string, m claims.Manifest, extraSensitiveRoutes []string, allowRemote bool) Outcome {
	var out Outcome
	base := strings.TrimRight(strings.TrimSpace(m.BaseURL), "/")
	if base == "" {
		return out
	}

	if err := validateLoopbackTarget(base, allowRemote); err != nil {
		out.Results = append(out.Results, model.ProbeResult{
			Name:        "Target safety boundary check",
			Kind:        "health_pulse",
			Executed:    false,
			Passed:      false,
			Target:      base,
			SummaryEN:   "Skipped live probes: BaseURL must be localhost/127.0.0.1 unless --allow-remote-probe is set.",
			SummaryID:   "Uji jalan dilewati: BaseURL wajib localhost/127.0.0.1 kecuali opsi --allow-remote-probe diaktifkan.",
			RawEvidence: err.Error(),
		})
		return out
	}

	client := &http.Client{
		Timeout: 4 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	healthPath := strings.TrimSpace(m.HealthPath)
	if healthPath == "" {
		healthPath = "/"
	}
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}

	// 1. Check whether local server is reachable; if not and StartCommand is configured, auto-boot it cleanly!
	resp, err := client.Get(base + healthPath)
	var stopBooted func()
	if err != nil && strings.TrimSpace(m.StartCommand) != "" {
		cmd, stopFn := buildShellCommand(workDir, m.StartCommand)
		if startErr := cmd.Start(); startErr == nil {
			stopBooted = stopFn
			out.AutoStarted = true
			deadline := time.Now().Add(12 * time.Second)
			for time.Now().Before(deadline) {
				time.Sleep(150 * time.Millisecond)
				if tryResp, tryErr := client.Get(base + healthPath); tryErr == nil {
					resp = tryResp
					err = nil
					break
				}
			}
		}
	}
	if stopBooted != nil {
		defer stopBooted()
	}

	if err != nil {
		out.Results = append(out.Results, model.ProbeResult{
			Name:        "Local application server pulse",
			Kind:        "health_pulse",
			Executed:    false,
			Passed:      false,
			Target:      base + healthPath,
			SummaryEN:   "Application server was not running locally during inspection; runtime behavior remains unproven.",
			SummaryID:   "Server aplikasi belum menyala saat diperiksa; perilaku jalan nyata belum teruji.",
			RawEvidence: err.Error(),
		})
		return out
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()

	out.AnyExecuted = true
	out.AllExecuted = true
	bootNoteEN := ""
	bootNoteID := ""
	if out.AutoStarted {
		bootNoteEN = " (auto-started via start_command)"
		bootNoteID = " (dinyalakan otomatis lewat start_command)"
	}
	out.Results = append(out.Results, model.ProbeResult{
		Name:          "Local application server pulse",
		Kind:          "health_pulse",
		Executed:      true,
		Passed:        resp.StatusCode < 500,
		Target:        base + healthPath,
		HTTPStatusGot: resp.StatusCode,
		SummaryEN:     fmt.Sprintf("Server responded live on %s (HTTP %d)%s.", healthPath, resp.StatusCode, bootNoteEN),
		SummaryID:     fmt.Sprintf("Server merespons aktif pada %s (HTTP %d)%s.", healthPath, resp.StatusCode, bootNoteID),
		RawEvidence:   fmt.Sprintf("GET %s -> %d", base+healthPath, resp.StatusCode),
	})

	// 2. Ghost-Wiring & Real Persistence Probes (SQLite Row-Delta, HTTP Read-Back for Postgres/Supabase/MySQL, or Both)
	for _, pc := range m.Persistence {
		res, finding := runPersistenceProbe(workDir, client, base, pc)
		out.Results = append(out.Results, res)
		if !res.Executed {
			out.AllExecuted = false
		}
		if finding != nil {
			out.FatalFindings = append(out.FatalFindings, *finding)
		}
	}

	// Merge declared protected routes with any undeclared sensitive routes discovered via Axoloom
	routeMap := map[string]claims.ProtectedRouteClaim{}
	for _, pr := range m.ProtectedRoutes {
		key := strings.ToUpper(pr.Method) + " " + pr.Path
		routeMap[key] = pr
	}
	for _, r := range extraSensitiveRoutes {
		key := "GET " + r
		if _, exists := routeMap[key]; !exists {
			routeMap[key] = claims.ProtectedRouteClaim{
				Name:   "Auto-discovered sensitive endpoint (" + r + ")",
				Method: "GET",
				Path:   r,
			}
		}
	}

	// 3. Unauthenticated Stranger Door Probes
	for _, pr := range routeMap {
		res, finding := runStrangerDoorProbe(client, base, pr)
		out.Results = append(out.Results, res)
		if finding != nil {
			out.FatalFindings = append(out.FatalFindings, *finding)
		}
	}

	// 4. Negative Quantity / Price Tamper Probes
	for _, mt := range m.MoneyTampers {
		res, finding := runMoneyTamperProbe(workDir, client, base, mt)
		out.Results = append(out.Results, res)
		if finding != nil {
			out.FatalFindings = append(out.FatalFindings, *finding)
		}
	}

	// 5. Real Headless Browser Screen Probes (375px phone viewport blank/crash/NaN detection)
	screenRes, screenFindings := runHeadlessScreenProbes(base, m.ScreenRoutes)
	out.Results = append(out.Results, screenRes...)
	out.FatalFindings = append(out.FatalFindings, screenFindings...)

	return out
}

func runPersistenceProbe(workDir string, client *http.Client, base string, pc claims.PersistenceClaim) (model.ProbeResult, *model.Finding) {
	path := ensureLeadingSlash(pc.Path)
	targetURL := base + path

	hasSQLite := strings.TrimSpace(pc.SQLitePath) != "" && strings.TrimSpace(pc.Table) != ""
	hasReadBack := strings.TrimSpace(pc.ReadBackPath) != ""

	if !hasSQLite && !hasReadBack {
		return model.ProbeResult{
			Name:        pc.Name,
			Kind:        "db_persistence",
			Executed:    false,
			Passed:      false,
			Target:      targetURL,
			SummaryEN:   "Persistence claim requires either sqlite_path+table or read_back_path.",
			SummaryID:   "Klaim simpan data membutuhkan sqlite_path+table atau read_back_path.",
			RawEvidence: "missing sqlite_path and read_back_path",
		}, nil
	}

	var rowsBefore int
	var dbFull string
	if hasSQLite {
		dbFull = resolveWorkspacePath(workDir, pc.SQLitePath)
		c, err := countSQLiteRows(dbFull, pc.Table)
		if err != nil {
			return model.ProbeResult{
				Name:        pc.Name,
				Kind:        "db_persistence",
				Executed:    false,
				Passed:      false,
				Target:      targetURL,
				SummaryEN:   fmt.Sprintf("Could not inspect SQLite table %s before action: %v", pc.Table, err),
				SummaryID:   fmt.Sprintf("Gagal membaca tabel database %s sebelum aksi: %v", pc.Table, err),
				RawEvidence: err.Error(),
			}, nil
		}
		rowsBefore = c
	}

	var rbBeforeCount int
	var rbBeforeBody string
	if hasReadBack {
		body, cnt, err := fetchReadBackState(client, base+ensureLeadingSlash(pc.ReadBackPath), pc.ReadHeaders)
		if err != nil {
			return model.ProbeResult{
				Name:        pc.Name,
				Kind:        "db_persistence",
				Executed:    false,
				Passed:      false,
				Target:      targetURL,
				SummaryEN:   "Could not read baseline state from read_back_path before action.",
				SummaryID:   "Gagal membaca data awal dari read_back_path sebelum aksi.",
				RawEvidence: err.Error(),
			}, nil
		}
		rbBeforeBody = body
		rbBeforeCount = cnt
	}

	method := strings.ToUpper(strings.TrimSpace(pc.Method))
	if method == "" {
		method = "POST"
	}
	req, err := http.NewRequestWithContext(context.Background(), method, targetURL, bytes.NewBufferString(pc.PayloadJSON))
	if err != nil {
		return model.ProbeResult{Name: pc.Name, Kind: "db_persistence", Target: targetURL, RawEvidence: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range pc.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return model.ProbeResult{
			Name:        pc.Name,
			Kind:        "db_persistence",
			Executed:    false,
			Passed:      false,
			Target:      targetURL,
			SummaryEN:   "Request failed to reach local endpoint.",
			SummaryID:   "Permintaan gagal menjangkau endpoint lokal.",
			RawEvidence: err.Error(),
		}, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	_ = resp.Body.Close()

	maxStatus := pc.ExpectStatusMax
	if maxStatus <= 0 {
		maxStatus = 299
	}
	minDelta := pc.ExpectMinDelta
	if minDelta <= 0 {
		minDelta = 1
	}

	statusOK := resp.StatusCode >= 200 && resp.StatusCode <= maxStatus
	sqliteOK := true
	rowsAfter := rowsBefore
	if hasSQLite {
		if after, aerr := countSQLiteRows(dbFull, pc.Table); aerr == nil {
			rowsAfter = after
		}
		sqliteOK = (rowsAfter - rowsBefore) >= minDelta
	}

	readBackOK := true
	rbAfterCount := rbBeforeCount
	var rbEvidence string
	if hasReadBack {
		afterBody, afterCnt, rerr := fetchReadBackState(client, base+ensureLeadingSlash(pc.ReadBackPath), pc.ReadHeaders)
		if rerr != nil {
			readBackOK = false
			rbEvidence = "read-back error"
		} else {
			rbAfterCount = afterCnt
			deltaRB := rbAfterCount - rbBeforeCount
			marker := strings.TrimSpace(pc.ExpectReadSubstring)
			if marker != "" {
				hadBefore := strings.Contains(rbBeforeBody, marker)
				hasAfter := strings.Contains(afterBody, marker)
				readBackOK = hasAfter && (!hadBefore || deltaRB >= minDelta)
				rbEvidence = fmt.Sprintf("GET %s items: %d -> %d (marker %s=%t)", ensureLeadingSlash(pc.ReadBackPath), rbBeforeCount, rbAfterCount, marker, hasAfter)
			} else {
				readBackOK = deltaRB >= minDelta
				rbEvidence = fmt.Sprintf("GET %s items: %d -> %d (delta %+d)", ensureLeadingSlash(pc.ReadBackPath), rbBeforeCount, rbAfterCount, deltaRB)
			}
		}
	}

	passed := statusOK && sqliteOK && readBackOK

	var rawParts []string
	rawParts = append(rawParts, fmt.Sprintf("HTTP %d", resp.StatusCode))
	if hasSQLite {
		rawParts = append(rawParts, fmt.Sprintf("%s rows: %d -> %d (delta %+d)", pc.Table, rowsBefore, rowsAfter, rowsAfter-rowsBefore))
	}
	if hasReadBack {
		rawParts = append(rawParts, rbEvidence)
	}
	rawStr := strings.Join(rawParts, " | ")

	storeLabel := pc.Table
	if storeLabel == "" {
		storeLabel = ensureLeadingSlash(pc.ReadBackPath)
	}

	res := model.ProbeResult{
		Name:          pc.Name,
		Kind:          "db_persistence",
		Executed:      true,
		Passed:        passed,
		Target:        fmt.Sprintf("%s %s -> %s", method, path, storeLabel),
		HTTPStatusGot: resp.StatusCode,
		RowsBefore:    rowsBefore,
		RowsAfter:     rowsAfter,
		RawEvidence:   rawStr,
	}

	if passed {
		if hasSQLite {
			res.SummaryEN = fmt.Sprintf("Proven real save: table %s grew from %d to %d rows (HTTP %d).", pc.Table, rowsBefore, rowsAfter, resp.StatusCode)
			res.SummaryID = fmt.Sprintf("Terbukti tersimpan nyata: isi tabel %s bertambah dari %d menjadi %d baris (HTTP %d).", pc.Table, rowsBefore, rowsAfter, resp.StatusCode)
		} else {
			res.SummaryEN = fmt.Sprintf("Proven persistent save via read-back on %s (items %d -> %d, HTTP %d).", ensureLeadingSlash(pc.ReadBackPath), rbBeforeCount, rbAfterCount, resp.StatusCode)
			res.SummaryID = fmt.Sprintf("Terbukti tersimpan permanen saat dibaca ulang di %s (data %d menjadi %d, HTTP %d).", ensureLeadingSlash(pc.ReadBackPath), rbBeforeCount, rbAfterCount, resp.StatusCode)
		}
		return res, nil
	}

	res.SummaryEN = fmt.Sprintf("FAILED persistence proof on %s (%s).", storeLabel, rawStr)
	res.SummaryID = fmt.Sprintf("GAGAL bukti simpan permanen pada %s (%s).", storeLabel, rawStr)

	finding := &model.Finding{
		ID:                "RUNTIME_GHOST_PERSISTENCE",
		Track:             model.TrackFatalBusinessRisk,
		Module:            model.ModuleLivePersistence,
		TitleEN:           fmt.Sprintf("Action (%s) did not persist durable data to %s", pc.Name, storeLabel),
		TitleID:           fmt.Sprintf("Fitur (%s) tidak benar-benar menyimpan data permanen ke %s", pc.Name, storeLabel),
		BusinessImpactEN:  fmt.Sprintf("Submitting the action returned HTTP %d, yet durable storage check failed (%s). Customer orders or records will vanish.", resp.StatusCode, rawStr),
		BusinessImpactID:  fmt.Sprintf("Saat dikirim server menjawab HTTP %d, namun pemeriksaan simpan data gagal (%s). Pesanan atau isian pelanggan akan hilang.", resp.StatusCode, rawStr),
		EvidenceSnippet:   rawStr,
		RepairInstruction: fmt.Sprintf("Fix %s %s so it commits durable records to your database (%s) before returning 2xx.", method, path, storeLabel),
	}
	return res, finding
}

func fetchReadBackState(client *http.Client, fullURL string, headers map[string]string) (string, int, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fullURL, nil)
	if err != nil {
		return "", 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		return "", 0, fmt.Errorf("read-back HTTP %d", resp.StatusCode)
	}
	return string(raw), countJSONItems(raw), nil
}

func countJSONItems(raw []byte) int {
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0
	}
	switch v := parsed.(type) {
	case []any:
		return len(v)
	case map[string]any:
		for _, key := range []string{"items", "data", "orders", "records", "results", "rows"} {
			if arr, ok := v[key].([]any); ok {
				return len(arr)
			}
		}
		return len(v)
	default:
		return 0
	}
}

func runStrangerDoorProbe(client *http.Client, base string, pr claims.ProtectedRouteClaim) (model.ProbeResult, *model.Finding) {
	path := ensureLeadingSlash(pr.Path)
	targetURL := base + path
	method := strings.ToUpper(strings.TrimSpace(pr.Method))
	if method == "" {
		method = "GET"
	}
	req, err := http.NewRequestWithContext(context.Background(), method, targetURL, nil)
	if err != nil {
		return model.ProbeResult{Name: pr.Name, Kind: "stranger_door", Target: targetURL, RawEvidence: err.Error()}, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return model.ProbeResult{
			Name:        pr.Name,
			Kind:        "stranger_door",
			Executed:    false,
			Passed:      false,
			Target:      targetURL,
			SummaryEN:   "Could not reach protected route for unauthenticated test.",
			SummaryID:   "Gagal menghubungi endpoint privat untuk uji akses tanpa login.",
			RawEvidence: err.Error(),
		}, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	_ = resp.Body.Close()

	locked := resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden ||
		(strings.HasPrefix(pr.Name, "Auto-discovered") && resp.StatusCode == http.StatusMethodNotAllowed)

	res := model.ProbeResult{
		Name:          pr.Name,
		Kind:          "stranger_door",
		Executed:      true,
		Passed:        locked,
		Target:        fmt.Sprintf("%s %s (no login)", method, path),
		HTTPStatusGot: resp.StatusCode,
		RawEvidence:   fmt.Sprintf("Unauthenticated %s %s -> HTTP %d", method, path, resp.StatusCode),
	}

	if locked {
		res.SummaryEN = fmt.Sprintf("Door locked: unauthenticated stranger blocked with HTTP %d on %s.", resp.StatusCode, path)
		res.SummaryID = fmt.Sprintf("Pintu terkunci rapat: akses orang asing tanpa login ditolak dengan HTTP %d pada %s.", resp.StatusCode, path)
		return res, nil
	}

	res.SummaryEN = fmt.Sprintf("UNLOCKED DOOR: unauthenticated request to %s returned HTTP %d instead of 401/403!", path, resp.StatusCode)
	res.SummaryID = fmt.Sprintf("PINTU TERBUKA TANPA LOGIN: akses tanpa login ke %s membalas HTTP %d (seharusnya ditolak 401/403)!", path, resp.StatusCode)

	finding := &model.Finding{
		ID:                "RUNTIME_UNLOCKED_DOOR",
		Track:             model.TrackFatalBusinessRisk,
		Module:            model.ModuleDoorLocks,
		TitleEN:           fmt.Sprintf("Private endpoint %s opened for an unlogged stranger (HTTP %d)", path, resp.StatusCode),
		TitleID:           fmt.Sprintf("Halaman/API privat %s bisa dibuka orang asing tanpa login (HTTP %d)", path, resp.StatusCode),
		BusinessImpactEN:  fmt.Sprintf("AxoProof knocked on %s without logging in and received HTTP %d. Customer or business records are exposed.", path, resp.StatusCode),
		BusinessImpactID:  fmt.Sprintf("AxoProof mencoba masuk ke %s tanpa login dan justru diberi akses HTTP %d. Data pelanggan atau bisnis bisa bocor ke publik.", path, resp.StatusCode),
		EvidenceSnippet:   res.RawEvidence,
		RepairInstruction: fmt.Sprintf("Enforce authentication check returning HTTP 401/403 for unauthenticated requests on %s %s.", method, path),
	}
	return res, finding
}

func runMoneyTamperProbe(workDir string, client *http.Client, base string, mt claims.MoneyTamperClaim) (model.ProbeResult, *model.Finding) {
	path := ensureLeadingSlash(mt.Path)
	targetURL := base + path
	method := strings.ToUpper(strings.TrimSpace(mt.Method))
	if method == "" {
		method = "POST"
	}

	hasDB := strings.TrimSpace(mt.SQLitePath) != "" && strings.TrimSpace(mt.Table) != ""
	var rowsBefore int
	if hasDB {
		rowsBefore, _ = countSQLiteRows(resolveWorkspacePath(workDir, mt.SQLitePath), mt.Table)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, targetURL, bytes.NewBufferString(mt.TamperedBodyJSON))
	if err != nil {
		return model.ProbeResult{Name: mt.Name, Kind: "negative_tamper", Target: targetURL, RawEvidence: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range mt.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return model.ProbeResult{
			Name:        mt.Name,
			Kind:        "negative_tamper",
			Executed:    false,
			Passed:      false,
			Target:      targetURL,
			SummaryEN:   "Could not reach transaction endpoint for tamper probe.",
			SummaryID:   "Gagal menghubungi endpoint transaksi untuk uji manipulasi angka.",
			RawEvidence: err.Error(),
		}, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	_ = resp.Body.Close()

	rowsAfter := rowsBefore
	if hasDB {
		if after, aerr := countSQLiteRows(resolveWorkspacePath(workDir, mt.SQLitePath), mt.Table); aerr == nil {
			rowsAfter = after
		}
	}

	rejected := resp.StatusCode >= 400 && resp.StatusCode < 500 && rowsAfter == rowsBefore
	res := model.ProbeResult{
		Name:          mt.Name,
		Kind:          "negative_tamper",
		Executed:      true,
		Passed:        rejected,
		Target:        fmt.Sprintf("%s %s (tampered payload)", method, path),
		HTTPStatusGot: resp.StatusCode,
		RowsBefore:    rowsBefore,
		RowsAfter:     rowsAfter,
		RawEvidence:   fmt.Sprintf("Tamper %s -> HTTP %d (rows %d -> %d)", path, resp.StatusCode, rowsBefore, rowsAfter),
	}

	if rejected {
		res.SummaryEN = fmt.Sprintf("Cash register defended: tampered quantity/price rejected with HTTP %d and 0 rows written.", resp.StatusCode)
		res.SummaryID = fmt.Sprintf("Kasir aman: percobaan manipulasi harga/jumlah minus ditolak dengan HTTP %d dan 0 data masuk.", resp.StatusCode)
		return res, nil
	}

	res.SummaryEN = fmt.Sprintf("CASH REGISTER EXPLOITED: tampered payload accepted with HTTP %d!", resp.StatusCode)
	res.SummaryID = fmt.Sprintf("CELAH KASIR TEMBUS: input jumlah minus atau manipulasi harga lolos dengan HTTP %d!", resp.StatusCode)

	finding := &model.Finding{
		ID:                "RUNTIME_MONEY_TAMPER_ACCEPTED",
		Track:             model.TrackFatalBusinessRisk,
		Module:            model.ModuleCashRegister,
		TitleEN:           fmt.Sprintf("Checkout accepted negative quantity or tampered price on %s (HTTP %d)", path, resp.StatusCode),
		TitleID:           fmt.Sprintf("Kasir meloloskan pesanan jumlah minus atau harga dimanipulasi pada %s (HTTP %d)", path, resp.StatusCode),
		BusinessImpactEN:  fmt.Sprintf("Sending negative quantity or tampered price to %s succeeded with HTTP %d. Buyers can manipulate totals or trigger negative charges.", path, resp.StatusCode),
		BusinessImpactID:  fmt.Sprintf("Pengiriman jumlah minus atau manipulasi harga ke %s justru diterima dengan HTTP %d. Pembeli nakal bisa mengakali total bayar.", path, resp.StatusCode),
		EvidenceSnippet:   res.RawEvidence,
		RepairInstruction: fmt.Sprintf("Validate quantity > 0, ignore client-supplied price fields, and return HTTP 400/422 on invalid inputs in %s %s.", method, path),
	}
	return res, finding
}

func countSQLiteRows(dbPath, table string) (int, error) {
	if !validSQLIdent.MatchString(table) {
		return 0, fmt.Errorf("unsafe SQLite table identifier %q", table)
	}
	if _, err := os.Stat(dbPath); err != nil {
		return 0, fmt.Errorf("sqlite file not found at %s", dbPath)
	}
	dsn := dbPath + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func ensureLeadingSlash(p string) string {
	p = strings.TrimSpace(p)
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func resolveWorkspacePath(workDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(workDir, p)
}

func validateLoopbackTarget(rawURL string, allowRemote bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if allowRemote {
		return nil
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("host %q is not loopback (127.0.0.1/localhost)", host)
}
