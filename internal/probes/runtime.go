package probes

import (
	"bytes"
	"context"
	"database/sql"
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
}

// RunAll executes counterfactual HTTP and SQLite row-delta probes when BaseURL is configured and reachable.
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

	// 1. Check whether local server is actually reachable
	resp, err := client.Get(base + healthPath)
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
	out.Results = append(out.Results, model.ProbeResult{
		Name:          "Local application server pulse",
		Kind:          "health_pulse",
		Executed:      true,
		Passed:        resp.StatusCode < 500,
		Target:        base + healthPath,
		HTTPStatusGot: resp.StatusCode,
		SummaryEN:     fmt.Sprintf("Server responded live on %s (HTTP %d).", healthPath, resp.StatusCode),
		SummaryID:     fmt.Sprintf("Server merespons aktif pada %s (HTTP %d).", healthPath, resp.StatusCode),
		RawEvidence:   fmt.Sprintf("GET %s -> %d", base+healthPath, resp.StatusCode),
	})

	// 2. Ghost-Wiring & Real SQLite Row-Delta Probes
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

	// Merge declared protected routes with any undeclared sensitive routes discovered via Axoloom/static scan
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

	return out
}

func runPersistenceProbe(workDir string, client *http.Client, base string, pc claims.PersistenceClaim) (model.ProbeResult, *model.Finding) {
	path := pc.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	targetURL := base + path
	dbFull := resolveWorkspacePath(workDir, pc.SQLitePath)

	rowsBefore, err := countSQLiteRows(dbFull, pc.Table)
	if err != nil {
		return model.ProbeResult{
			Name:        pc.Name,
			Kind:        "db_persistence",
			Executed:    false,
			Passed:      false,
			Target:      targetURL,
			SummaryEN:   fmt.Sprintf("Could not inspect SQLite table %q before action: %v", pc.Table, err),
			SummaryID:   fmt.Sprintf("Gagal membaca tabel database %q sebelum aksi: %v", pc.Table, err),
			RawEvidence: err.Error(),
		}, nil
	}

	req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(pc.Method), targetURL, bytes.NewBufferString(pc.PayloadJSON))
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

	rowsAfter, afterErr := countSQLiteRows(dbFull, pc.Table)
	if afterErr != nil {
		rowsAfter = rowsBefore
	}
	delta := rowsAfter - rowsBefore
	maxStatus := pc.ExpectStatusMax
	if maxStatus <= 0 {
		maxStatus = 299
	}
	minDelta := pc.ExpectMinDelta
	if minDelta <= 0 {
		minDelta = 1
	}
	passed := resp.StatusCode >= 200 && resp.StatusCode <= maxStatus && delta >= minDelta

	res := model.ProbeResult{
		Name:          pc.Name,
		Kind:          "db_persistence",
		Executed:      true,
		Passed:        passed,
		Target:        fmt.Sprintf("%s %s -> table %s", strings.ToUpper(pc.Method), path, pc.Table),
		HTTPStatusGot: resp.StatusCode,
		RowsBefore:    rowsBefore,
		RowsAfter:     rowsAfter,
		RawEvidence:   fmt.Sprintf("HTTP %d | %s rows: %d -> %d (delta %+d)", resp.StatusCode, pc.Table, rowsBefore, rowsAfter, delta),
	}

	if passed {
		res.SummaryEN = fmt.Sprintf("Proven real save: table %s grew from %d to %d rows (HTTP %d).", pc.Table, rowsBefore, rowsAfter, resp.StatusCode)
		res.SummaryID = fmt.Sprintf("Terbukti tersimpan nyata: isi tabel %s bertambah dari %d menjadi %d baris (HTTP %d).", pc.Table, rowsBefore, rowsAfter, resp.StatusCode)
		return res, nil
	}

	res.SummaryEN = fmt.Sprintf("FAILED persistence proof: HTTP %d returned, but table %s went from %d to %d rows (needed +%d).", resp.StatusCode, pc.Table, rowsBefore, rowsAfter, minDelta)
	res.SummaryID = fmt.Sprintf("GAGAL bukti simpan: server menjawab HTTP %d, tetapi isi tabel %s tetap %d ke %d baris (wajib bertambah minimal +%d).", resp.StatusCode, pc.Table, rowsBefore, rowsAfter, minDelta)

	finding := &model.Finding{
		ID:                "RUNTIME_GHOST_PERSISTENCE",
		Track:             model.TrackFatalBusinessRisk,
		Module:            model.ModuleLivePersistence,
		TitleEN:           fmt.Sprintf("Action (%s) claimed success without saving to database (%s)", pc.Name, pc.Table),
		TitleID:           fmt.Sprintf("Fitur (%s) terlihat berhasil tetapi tidak menyimpan data ke tabel %s", pc.Name, pc.Table),
		BusinessImpactEN:  fmt.Sprintf("Pressing submit returned HTTP %d, yet table %s stayed at %d rows. Customer orders or form submissions will vanish.", resp.StatusCode, pc.Table, rowsAfter),
		BusinessImpactID:  fmt.Sprintf("Saat dikirim, server menjawab HTTP %d, padahal isi tabel %s tetap %d baris. Pesanan atau data pelanggan akan hilang begitu saja.", resp.StatusCode, pc.Table, rowsAfter),
		EvidenceSnippet:   res.RawEvidence,
		RepairInstruction: fmt.Sprintf("Fix handler %s %s so it commits a durable INSERT into SQLite table %s within a transaction before returning 2xx.", strings.ToUpper(pc.Method), path, pc.Table),
	}
	return res, finding
}

func runStrangerDoorProbe(client *http.Client, base string, pr claims.ProtectedRouteClaim) (model.ProbeResult, *model.Finding) {
	path := pr.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	targetURL := base + path
	req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(pr.Method), targetURL, nil)
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
		Target:        fmt.Sprintf("%s %s (no login)", strings.ToUpper(pr.Method), path),
		HTTPStatusGot: resp.StatusCode,
		RawEvidence:   fmt.Sprintf("Unauthenticated %s %s -> HTTP %d", strings.ToUpper(pr.Method), path, resp.StatusCode),
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
		RepairInstruction: fmt.Sprintf("Enforce authentication check returning HTTP 401/403 for unauthenticated requests on %s %s.", strings.ToUpper(pr.Method), path),
	}
	return res, finding
}

func runMoneyTamperProbe(workDir string, client *http.Client, base string, mt claims.MoneyTamperClaim) (model.ProbeResult, *model.Finding) {
	path := mt.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	targetURL := base + path

	hasDB := strings.TrimSpace(mt.SQLitePath) != "" && strings.TrimSpace(mt.Table) != ""
	var rowsBefore int
	if hasDB {
		rowsBefore, _ = countSQLiteRows(resolveWorkspacePath(workDir, mt.SQLitePath), mt.Table)
	}

	req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(mt.Method), targetURL, bytes.NewBufferString(mt.TamperedBodyJSON))
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
		Target:        fmt.Sprintf("%s %s (tampered payload)", strings.ToUpper(mt.Method), path),
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
		RepairInstruction: fmt.Sprintf("Validate quantity > 0, ignore client-supplied price fields, and return HTTP 400/422 on invalid inputs in %s %s.", strings.ToUpper(mt.Method), path),
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

