package insights

import "encoding/json"

type TransactionKind string
type DeviationClass string
type SampleReason string
type PolicyScope string
type Severity string
type FindingKind string
type AnomalyStatus string
type ViolationStatus string
type RuleMode string
type AnomalyResolution string
type BulkResolution string
type LearningMode string
type LearningConditionType string
type StorageHealthStatus string
type StorageDiskStatus string

type ListTransactionsParams struct {
	Namespace *string
	Service   *string
	Kind      *TransactionKind
}

type ListObservationsParams struct {
	SampleReason *SampleReason
}

type ListFindingsParams struct {
	WindowHours *int
	Service     *string
	Namespace   *string
	Status      *string
	Kind        *FindingKind
}

type ListAnomaliesParams struct {
	WindowHours *int
	Service     *string
	Namespace   *string
	Status      *string
}

type ListGuardrailViolationsParams struct {
	WindowHours *int
	Service     *string
	Namespace   *string
	Status      *string
}

type ServiceStat struct {
	Namespace        string `json:"namespace"`
	Service          string `json:"service"`
	TransactionCount int64  `json:"transaction_count"`
	Volume           int64  `json:"volume"`
	LastSeen         string `json:"last_seen"`
}

type ServiceProfile struct {
	Namespace string   `json:"namespace"`
	Service   string   `json:"service"`
	Callers   []string `json:"callers"`
	Callees   []string `json:"callees"`
	Egress    []string `json:"egress"`
	// Transactions are transaction identities in "KIND\toperation" format.
	Transactions []string `json:"transactions"`
}

type TransactionStat struct {
	ID          int64           `json:"id"`
	Service     string          `json:"service"`
	Namespace   string          `json:"namespace"`
	Operation   string          `json:"operation"`
	Kind        TransactionKind `json:"kind"`
	Volume      int64           `json:"volume"`
	LastSeen    string          `json:"last_seen"`
	HasBaseline *bool           `json:"has_baseline,omitempty"`
	Promoted    *bool           `json:"promoted,omitempty"`
}

type Transaction struct {
	ID        int64           `json:"id"`
	Service   string          `json:"service"`
	Namespace string          `json:"namespace"`
	Operation string          `json:"operation"`
	Kind      TransactionKind `json:"kind"`
}

type BaselineClass struct {
	TransactionID                int64           `json:"transaction_id"`
	Class                        DeviationClass  `json:"class"`
	Data                         json.RawMessage `json:"data,omitempty"`
	DataSchemaVersion            *int            `json:"data_schema_version,omitempty"`
	ObservationCount             int64           `json:"observation_count"`
	Promoted                     bool            `json:"promoted"`
	LearningStartedAt            *string         `json:"learning_started_at,omitempty"`
	LastChangedAt                *string         `json:"last_changed_at,omitempty"`
	ObservationCountAtLastChange *int64          `json:"observation_count_at_last_change,omitempty"`
}

type PromoteResult struct {
	TransactionID int64          `json:"transaction_id"`
	Class         DeviationClass `json:"class"`
	Promoted      bool           `json:"promoted"`
}

type ObservationSummary struct {
	TraceID       string       `json:"trace_id"`
	TransactionID int64        `json:"transaction_id"`
	ObservedAt    string       `json:"observed_at"`
	DurationMs    int64        `json:"duration_ms"`
	SampleReason  SampleReason `json:"sample_reason"`
}

type Observation struct {
	TraceID       string       `json:"trace_id"`
	TransactionID int64        `json:"transaction_id"`
	ObservedAt    string       `json:"observed_at"`
	DurationMs    int64        `json:"duration_ms"`
	SampleReason  SampleReason `json:"sample_reason"`
	RawOTLP       string       `json:"raw_otlp"`
}

type Policy struct {
	ID            int64               `json:"id"`
	Name          string              `json:"name"`
	Enabled       bool                `json:"enabled"`
	FireAtScore   int                 `json:"fire_at_score"`
	SignalWeights map[string]int      `json:"signal_weights,omitempty"`
	EnricherLists map[string][]string `json:"enricher_lists,omitempty"`
	Scope         PolicyScope         `json:"scope"`
	ScopeKey      string              `json:"scope_key"`
}

// LearningCondition is a single promotion criterion. JSON (un)marshaling is
// defined in conversions.go because the REST wire form uses camelCase field
// names — unlike the rest of the insights API, which is snake_case.
type LearningCondition struct {
	Type               LearningConditionType
	MinObservations    *int64
	MinDurationMinutes *int64
	StableObservations *int64
	StableMinutes      *int64
}

type LearningPolicy struct {
	Class      DeviationClass      `json:"class"`
	Mode       LearningMode        `json:"mode"`
	Conditions []LearningCondition `json:"conditions,omitempty"`
	MinMatches *int                `json:"min_matches,omitempty"`
	Scope      PolicyScope         `json:"scope"`
	ScopeKey   string              `json:"scope_key"`
}

type Finding struct {
	Kind          FindingKind `json:"kind"`
	Service       string      `json:"service"`
	Namespace     string      `json:"namespace"`
	Title         string      `json:"title"`
	Offending     *string     `json:"offending,omitempty"`
	Score         int         `json:"score"`
	Severity      Severity    `json:"severity"`
	Occurrences   int64       `json:"occurrences"`
	LastSeen      string      `json:"last_seen"`
	Status        string      `json:"status"`
	TransactionID *int64      `json:"transaction_id,omitempty"`
	Signature     *string     `json:"signature,omitempty"`
	ScopeKey      *string     `json:"scope_key,omitempty"`
	RuleKey       *string     `json:"rule_key,omitempty"`
}

type RiskSignal struct {
	Source       string  `json:"source"`
	Category     string  `json:"category"`
	Weight       int     `json:"weight"`
	Confidence   float64 `json:"confidence"`
	Contribution int     `json:"contribution"`
	Rationale    *string `json:"rationale,omitempty"`
	Mitre        *string `json:"mitre,omitempty"`
	Owasp        *string `json:"owasp,omitempty"`
}

type RiskAssessment struct {
	Score            int          `json:"score"`
	Severity         Severity     `json:"severity"`
	Correlated       bool         `json:"correlated"`
	CorrelationBonus int          `json:"correlation_bonus"`
	FireAtScore      int          `json:"fire_at_score"`
	Categories       []string     `json:"categories,omitempty"`
	Signals          []RiskSignal `json:"signals"`
}

type AnomalySummary struct {
	TransactionID    int64            `json:"transaction_id"`
	Signature        string           `json:"signature"`
	Service          string           `json:"service"`
	Namespace        string           `json:"namespace"`
	Operation        *string          `json:"operation,omitempty"`
	Kind             *TransactionKind `json:"kind,omitempty"`
	TriggeredClasses []DeviationClass `json:"triggered_classes"`
	Offending        *string          `json:"offending,omitempty"`
	PolicyID         *int64           `json:"policy_id,omitempty"`
	Occurrences      int64            `json:"occurrences"`
	MaxScore         int              `json:"max_score"`
	Severity         Severity         `json:"severity"`
	FirstSeen        *string          `json:"first_seen,omitempty"`
	LastSeen         *string          `json:"last_seen,omitempty"`
	LastTraceID      *string          `json:"last_trace_id,omitempty"`
	Status           AnomalyStatus    `json:"status"`
}

type AnomalyIssue struct {
	TransactionID    int64            `json:"transaction_id"`
	Signature        string           `json:"signature"`
	Service          string           `json:"service"`
	Namespace        string           `json:"namespace"`
	Operation        *string          `json:"operation,omitempty"`
	Kind             *TransactionKind `json:"kind,omitempty"`
	TriggeredClasses []DeviationClass `json:"triggered_classes"`
	Offending        *string          `json:"offending,omitempty"`
	Evidence         json.RawMessage  `json:"evidence"`
	PolicyID         *int64           `json:"policy_id,omitempty"`
	Occurrences      int64            `json:"occurrences"`
	MaxScore         int              `json:"max_score"`
	Severity         Severity         `json:"severity"`
	FirstSeen        *string          `json:"first_seen,omitempty"`
	LastSeen         *string          `json:"last_seen,omitempty"`
	LastTraceID      *string          `json:"last_trace_id,omitempty"`
	Status           AnomalyStatus    `json:"status"`
	Risk             *RiskAssessment  `json:"risk,omitempty"`
}

type AnomalyRef struct {
	TransactionID int64  `json:"transaction_id"`
	Signature     string `json:"signature"`
}

type BulkAnomalyRequest struct {
	Resolution BulkResolution `json:"resolution"`
	Items      []AnomalyRef   `json:"items"`
}

type BulkResolveResult struct {
	Resolution string `json:"resolution"`
	Resolved   int    `json:"resolved"`
}

type GuardrailRule struct {
	Key       string   `json:"key"`
	Label     string   `json:"label"`
	Mode      RuleMode `json:"mode"`
	Allowlist []string `json:"allowlist,omitempty"`
}

type Guardrail struct {
	Scope    PolicyScope     `json:"scope"`
	ScopeKey string          `json:"scope_key"`
	Rules    []GuardrailRule `json:"rules"`
}

type GuardrailViolation struct {
	Service     string          `json:"service"`
	Namespace   string          `json:"namespace"`
	ScopeKey    string          `json:"scope_key"`
	RuleKey     string          `json:"rule_key"`
	RuleLabel   string          `json:"rule_label"`
	Offending   *string         `json:"offending,omitempty"`
	Occurrences int64           `json:"occurrences"`
	MaxScore    int             `json:"max_score"`
	Severity    Severity        `json:"severity"`
	LastSeen    *string         `json:"last_seen,omitempty"`
	Status      ViolationStatus `json:"status"`
}

type ViolationActionRequest struct {
	ScopeKey  string `json:"scope_key"`
	RuleKey   string `json:"rule_key"`
	Offending string `json:"offending"`
}

type GuardrailSeedRequest struct {
	ScopeKey string              `json:"scope_key"`
	Mode     *RuleMode           `json:"mode,omitempty"`
	Items    map[string][]string `json:"items"`
}

type CatalogClass struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Description   *string `json:"description,omitempty"`
	Category      string  `json:"category"`
	CategoryLabel *string `json:"category_label,omitempty"`
	Weight        int     `json:"weight"`
	Mitre         *string `json:"mitre,omitempty"`
	Owasp         *string `json:"owasp,omitempty"`
}

type EnricherList struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Hint    *string  `json:"hint,omitempty"`
	Default []string `json:"default,omitempty"`
}

type CatalogEnricher struct {
	Source      string        `json:"source"`
	ParentClass string        `json:"parent_class"`
	Category    string        `json:"category"`
	Weight      int           `json:"weight"`
	Label       string        `json:"label"`
	Description *string       `json:"description,omitempty"`
	List        *EnricherList `json:"list,omitempty"`
}

type CatalogCategory struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Base  int     `json:"base"`
	Mitre *string `json:"mitre,omitempty"`
	Owasp *string `json:"owasp,omitempty"`
}

type CatalogGuardrailRule struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
	Weight      int     `json:"weight"`
	Hint        *string `json:"hint,omitempty"`
}

type SeverityBand struct {
	Severity Severity `json:"severity"`
	MinScore int      `json:"min_score"`
}

type Catalog struct {
	DeviationClasses []CatalogClass         `json:"deviation_classes"`
	Enrichers        []CatalogEnricher      `json:"enrichers"`
	Categories       []CatalogCategory      `json:"categories"`
	GuardrailRules   []CatalogGuardrailRule `json:"guardrail_rules"`
	SeverityBands    []SeverityBand         `json:"severity_bands"`
	CorrelationBonus int                    `json:"correlation_bonus"`
	Tags             map[string]string      `json:"tags"`
}

type SystemSamplingSettings struct {
	ExamplesPerTransaction       int `json:"examples_per_transaction"`
	ExampleSampleIntervalSeconds int `json:"example_sample_interval_seconds"`
}

type SystemRetentionSettings struct {
	ObservationRetentionDays int `json:"observation_retention_days"`
}

type SystemFindingsSettings struct {
	DefaultWindowHours int `json:"default_window_hours"`
	MaxWindowHours     int `json:"max_window_hours"`
}

type SystemCapacitySettings struct {
	MaxResidentTransactions int `json:"max_resident_transactions"`
	MaxBaselineSetMembers   int `json:"max_baseline_set_members"`
}

type SystemWritebackSettings struct {
	FlushIntervalSeconds int `json:"flush_interval_seconds"`
}

type SystemSettings struct {
	Sampling  SystemSamplingSettings  `json:"sampling"`
	Retention SystemRetentionSettings `json:"retention"`
	Findings  SystemFindingsSettings  `json:"findings"`
	Capacity  SystemCapacitySettings  `json:"capacity"`
	Writeback SystemWritebackSettings `json:"writeback"`
}

type StorageConnection struct {
	Reachable     bool    `json:"reachable"`
	PingLatencyMs int64   `json:"ping_latency_ms"`
	Version       string  `json:"version"`
	Database      string  `json:"database"`
	Error         *string `json:"error,omitempty"`
}

type StorageDisk struct {
	InsightsBytes  int64             `json:"insights_bytes"`
	TotalBytes     int64             `json:"total_bytes"`
	UsedBytes      int64             `json:"used_bytes"`
	AvailableBytes int64             `json:"available_bytes"`
	UsedPercent    float64           `json:"used_percent"`
	Status         StorageDiskStatus `json:"status"`
}

type StorageMemory struct {
	ProcessRSSBytes  int64   `json:"process_rss_bytes"`
	CgroupUsedBytes  int64   `json:"cgroup_used_bytes"`
	CgroupTotalBytes int64   `json:"cgroup_total_bytes"`
	UptimeSeconds    float64 `json:"uptime_seconds"`
}

type StorageQuery struct {
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	User           string  `json:"user"`
	MemoryBytes    int64   `json:"memory_bytes"`
	Query          string  `json:"query"`
	FinishedAt     *string `json:"finished_at,omitempty"`
}

type StorageQueries struct {
	Active          int            `json:"active"`
	LongestSeconds  float64        `json:"longest_seconds"`
	PeakMemoryBytes int64          `json:"peak_memory_bytes"`
	Items           []StorageQuery `json:"items"`
	RecentHeaviest  []StorageQuery `json:"recent_heaviest"`
	RecentLatest    []StorageQuery `json:"recent_latest"`
}

type StorageTable struct {
	Name        string `json:"name"`
	Rows        int64  `json:"rows"`
	BytesOnDisk int64  `json:"bytes_on_disk"`
	ActiveParts int64  `json:"active_parts"`
}

type StorageWritePressure struct {
	UnfinishedMutations int64 `json:"unfinished_mutations"`
	ActiveParts         int64 `json:"active_parts"`
}

type StorageWriteback struct {
	FlushIntervalSeconds int     `json:"flush_interval_seconds"`
	EverFlushed          bool    `json:"ever_flushed"`
	LastFlushOK          bool    `json:"last_flush_ok"`
	LastFlushAt          *string `json:"last_flush_at,omitempty"`
	LastFlushError       *string `json:"last_flush_error,omitempty"`
}

type StorageHealth struct {
	CheckedAt     string               `json:"checked_at"`
	Status        StorageHealthStatus  `json:"status"`
	Connection    StorageConnection    `json:"connection"`
	Disk          StorageDisk          `json:"disk"`
	Memory        StorageMemory        `json:"memory"`
	Queries       StorageQueries       `json:"queries"`
	Tables        []StorageTable       `json:"tables"`
	WritePressure StorageWritePressure `json:"write_pressure"`
	Writeback     StorageWriteback     `json:"writeback"`
}
