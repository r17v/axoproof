package axobridge

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/r17v/axoproof/internal/claims"
	"github.com/r17v/axoproof/internal/model"
	_ "modernc.org/sqlite"
)

var reSensitiveEndpoint = regexp.MustCompile(`(?i)^/api/(admin|customers|users|orders|invoices|patients|payments|wallets|transactions|ledger)\b`)

// Inspect connects to a local or ancestor .axoloom/axoloom.db if available, pulling real AST endpoints,
// SQL tables, and guardrails so AI coding agents cannot hide sensitive routes from AxoProof.
func Inspect(workDir string, declaredProtected []claims.ProtectedRouteClaim) model.AxoloomContext {
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return model.AxoloomContext{}
	}

	dbPath, axoRoot := locateAxoloomDB(absWork)
	if dbPath == "" {
		return model.AxoloomContext{}
	}

	dsn := dbPath + "?_pragma=busy_timeout(5000)&mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return model.AxoloomContext{}
	}
	defer db.Close()

	ctx := model.AxoloomContext{
		Connected: true,
		DBPath:    relOrBase(axoRoot, dbPath),
	}

	projectName := filepath.Base(absWork)
	pathLike := absWork + "%"

	// Count SQL tables in project scope (or whole DB if workspace root IS axoloom root)
	if absWork == axoRoot {
		_ = db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE kind = 'table'`).Scan(&ctx.IndexedSQLTables)
		_ = db.QueryRow(`SELECT COUNT(*) FROM guardrails`).Scan(&ctx.ActiveGuardrails)
	} else {
		_ = db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE kind = 'table' AND (LOWER(project) = LOWER(?) OR path LIKE ?)`, projectName, pathLike).Scan(&ctx.IndexedSQLTables)
		_ = db.QueryRow(`SELECT COUNT(*) FROM guardrails WHERE LOWER(project) = LOWER(?) OR source_path LIKE ?`, projectName, pathLike).Scan(&ctx.ActiveGuardrails)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM directives`).Scan(&ctx.ActiveDirectives)

	declaredSet := map[string]bool{}
	for _, p := range declaredProtected {
		declaredSet[strings.TrimSpace(p.Path)] = true
	}

	var rows *sql.Rows
	if absWork == axoRoot {
		rows, err = db.Query(`SELECT name FROM nodes WHERE kind IN ('endpoint', 'route')`)
	} else {
		rows, err = db.Query(`SELECT name FROM nodes WHERE kind IN ('endpoint', 'route') AND (LOWER(project) = LOWER(?) OR path LIKE ?)`, projectName, pathLike)
	}
	if err == nil && rows != nil {
		defer rows.Close()
		seen := map[string]bool{}
		for rows.Next() {
			var ep string
			if err := rows.Scan(&ep); err != nil {
				continue
			}
			route := extractPathFromEndpointName(ep)
			if route == "" || seen[route] {
				continue
			}
			seen[route] = true
			ctx.IndexedEndpoints++
			if reSensitiveEndpoint.MatchString(route) && !declaredSet[route] {
				ctx.UndeclaredRoutesFound = append(ctx.UndeclaredRoutesFound, route)
			}
		}
	}

	return ctx
}

// SyncBack writes the sealed verification outcome back to Axoloom handoff memory when requested.
func SyncBack(workDir string, receipt model.SealedReceipt) string {
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return ""
	}
	_, axoRoot := locateAxoloomDB(absWork)
	if axoRoot == "" {
		return ""
	}

	bin, err := exec.LookPath("axoloom")
	if err != nil {
		if home, herr := os.UserHomeDir(); herr == nil {
			cand := filepath.Join(home, ".local", "bin", "axoloom")
			if _, serr := os.Stat(cand); serr == nil {
				bin = cand
			}
		}
	}
	if bin == "" {
		return ""
	}

	summary := fmt.Sprintf("AxoProof [%s] on %s: %d fatal risks, %d polish items, %d runtime probes (seal %s)",
		receipt.Verdict, receipt.ProjectName, len(receipt.FatalRisks), len(receipt.PolishWarnings), len(receipt.RuntimeProbes), receipt.SealSHA256[:12])
	cmd := exec.Command(bin, "handoff", "--dir", axoRoot, "axoproof", summary)
	if err := cmd.Run(); err == nil {
		return filepath.Join(".axoloom", "handoff", "axoproof.md")
	}
	return ""
}

// ExpandBlastRadiusFiles queries Axoloom's edges/nodes graph to include 1-hop upstream caller files
// for any modified relative file path when running in --scope diff mode.
func ExpandBlastRadiusFiles(workDir string, relFiles map[string]bool) map[string]bool {
	if len(relFiles) == 0 {
		return relFiles
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return relFiles
	}
	dbPath, _ := locateAxoloomDB(absWork)
	if dbPath == "" {
		return relFiles
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&mode=ro")
	if err != nil {
		return relFiles
	}
	defer db.Close()

	expanded := map[string]bool{}
	for k := range relFiles {
		expanded[k] = true
	}

	for rel := range relFiles {
		absTarget := filepath.Join(absWork, filepath.FromSlash(rel))
		rows, qerr := db.Query(`
			SELECT DISTINCT e.path
			FROM nodes n
			JOIN edges e ON (e.dst_id = n.id OR e.dst_name = n.name)
			WHERE n.path = ? OR n.path LIKE ?
		`, absTarget, "%/"+rel)
		if qerr != nil || rows == nil {
			continue
		}
		for rows.Next() {
			var callerPath string
			if err := rows.Scan(&callerPath); err == nil && callerPath != "" {
				if callerRel, rerr := filepath.Rel(absWork, callerPath); rerr == nil && !strings.HasPrefix(callerRel, "..") {
					expanded[filepath.ToSlash(callerRel)] = true
				} else if !filepath.IsAbs(callerPath) {
					expanded[filepath.ToSlash(callerPath)] = true
				}
			}
		}
		_ = rows.Close()
	}
	return expanded
}

func locateAxoloomDB(startDir string) (string, string) {
	cur := startDir
	for i := 0; i < 3; i++ {
		for _, dbName := range []string{"index.db", "axoloom.db"} {
			cand := filepath.Join(cur, ".axoloom", dbName)
			if info, err := os.Stat(cand); err == nil && !info.IsDir() {
				return cand, cur
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return "", ""
}

func extractPathFromEndpointName(raw string) string {
	fields := strings.Fields(strings.TrimSpace(raw))
	for _, f := range fields {
		if strings.HasPrefix(f, "/") {
			return f
		}
	}
	if strings.HasPrefix(raw, "/") {
		return raw
	}
	return ""
}

func relOrBase(root, target string) string {
	if rel, err := filepath.Rel(root, target); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.Base(target)
}
