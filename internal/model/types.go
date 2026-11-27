package model

// VerdictStatus represents the top-level owner decision banner.
type VerdictStatus string

const (
	// VerdictSafeToTest is awarded ONLY when zero fatal defects exist AND live runtime probes succeeded.
	VerdictSafeToTest VerdictStatus = "SAFE_TO_TEST"
	// VerdictLocalOnlyManualCheck is awarded when zero fatal static defects exist, but live runtime or external services remain unproven.
	VerdictLocalOnlyManualCheck VerdictStatus = "LOCAL_ONLY_MANUAL_CHECK"
	// VerdictHoldRelease is triggered whenever a placebo behavior, money integrity flaw, or open auth door is found.
	VerdictHoldRelease VerdictStatus = "HOLD_RELEASE"
)

// FindingTrack separates fatal business/money/security defects from customer-facing polish & copy issues.
type FindingTrack string

const (
	TrackFatalBusinessRisk FindingTrack = "FATAL_BUSINESS_RISK"
	TrackCustomerPolish    FindingTrack = "CUSTOMER_POLISH"
)

// DomainModule identifies which of the 6 specialist inspector lenses caught an item.
type DomainModule string

const (
	ModuleNoPlacebo       DomainModule = "no_placebo"
	ModuleCashRegister    DomainModule = "cash_register"
	ModuleDoorLocks       DomainModule = "door_locks"
	ModuleLivePersistence DomainModule = "live_persistence"
	ModuleCustomerReady   DomainModule = "customer_ready"
)

// Finding holds both a zero-jargon owner explanation (EN + ID) and an exact machine repair brief for the AI agent.
type Finding struct {
	ID                string       `json:"id"`
	Track             FindingTrack `json:"track"`
	Module            DomainModule `json:"module"`
	File              string       `json:"file,omitempty"`
	Line              int          `json:"line,omitempty"`
	TitleEN           string       `json:"title_en"`
	TitleID           string       `json:"title_id"`
	BusinessImpactEN  string       `json:"business_impact_en"`
	BusinessImpactID  string       `json:"business_impact_id"`
	EvidenceSnippet   string       `json:"evidence_snippet,omitempty"`
	RepairInstruction string       `json:"repair_instruction"`
}

// ProbeResult records concrete, falsifiable runtime measurements executed out-of-band by the CLI.
type ProbeResult struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"` // db_persistence | stranger_door | negative_tamper | health_pulse
	Passed        bool   `json:"passed"`
	Executed      bool   `json:"executed"`
	Target        string `json:"target"`
	SummaryEN     string `json:"summary_en"`
	SummaryID     string `json:"summary_id"`
	RawEvidence   string `json:"raw_evidence"`
	RowsBefore    int    `json:"rows_before,omitempty"`
	RowsAfter     int    `json:"rows_after,omitempty"`
	HTTPStatusGot int    `json:"http_status_got,omitempty"`
}

// BlindspotItem explicitly tells the owner what still requires a 2-minute human eye check.
type BlindspotItem struct {
	TitleEN string `json:"title_en"`
	TitleID string `json:"title_id"`
	WhyEN   string `json:"why_en"`
	WhyID   string `json:"why_id"`
}

// AxoloomContext captures live cross-tool collaboration with .axoloom/axoloom.db when present.
type AxoloomContext struct {
	Connected             bool     `json:"connected"`
	DBPath                string   `json:"db_path,omitempty"`
	IndexedEndpoints      int      `json:"indexed_endpoints"`
	IndexedSQLTables      int      `json:"indexed_sql_tables"`
	ActiveGuardrails      int      `json:"active_guardrails"`
	ActiveDirectives      int      `json:"active_directives"`
	UndeclaredRoutesFound []string `json:"undeclared_routes_found,omitempty"`
	WritebackPath         string   `json:"writeback_path,omitempty"`
}

// SealedReceipt is the tamper-evident verification artifact rendered directly by the AxoProof binary.
type SealedReceipt struct {
	SchemaVersion    string          `json:"schema_version"`
	ProjectName      string          `json:"project_name"`
	WorkspaceSummary string          `json:"workspace_summary"`
	GeneratedAtUTC   string          `json:"generated_at_utc"`
	GitCommit        string          `json:"git_commit"`
	GitDirtyFiles    int             `json:"git_dirty_files"`
	Verdict          VerdictStatus   `json:"verdict"`
	HeadlineEN       string          `json:"headline_en"`
	HeadlineID       string          `json:"headline_id"`
	SubheadlineEN    string          `json:"subheadline_en"`
	SubheadlineID    string          `json:"subheadline_id"`
	FilesScanned     int             `json:"files_scanned"`
	FatalRisks       []Finding       `json:"fatal_risks"`
	PolishWarnings   []Finding       `json:"polish_warnings"`
	RuntimeProbes    []ProbeResult   `json:"runtime_probes"`
	Blindspots       []BlindspotItem `json:"blindspots"`
	Axoloom          AxoloomContext  `json:"axoloom"`
	SealSHA256       string          `json:"seal_sha256"`
}
