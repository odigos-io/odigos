package insights

import "encoding/json"

type DeliverySettings struct {
	Muted                  bool     `json:"muted"`
	ExcludedIntegrationIds []string `json:"excluded_integration_ids"`
}

type IntegrationFieldSpec struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Help        string   `json:"help"`
	Placeholder string   `json:"placeholder"`
	Default     string   `json:"default"`
	Options     []string `json:"options"`
}

type IntegrationCapabilities struct {
	NeedsHeartbeat    bool `json:"needs_heartbeat"`
	SupportsResolve   bool `json:"supports_resolve"`
	ReturnsExternalID bool `json:"returns_external_id"`
	MaxBatch          int  `json:"max_batch"`
}

type IntegrationDescriptor struct {
	Type         string                  `json:"type"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description"`
	DocsURL      string                  `json:"docs_url"`
	Fields       []IntegrationFieldSpec  `json:"fields"`
	Capabilities IntegrationCapabilities `json:"capabilities"`
}

type IntegrationFilters struct {
	MinSeverity     *Severity     `json:"min_severity,omitempty"`
	Kinds           []FindingKind `json:"kinds"`
	NotifyOnResolve bool          `json:"notify_on_resolve"`
}

type IntegrationHealth struct {
	LastSuccessAt       string `json:"last_success_at"`
	LastFailureAt       string `json:"last_failure_at"`
	LastError           string `json:"last_error"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	Pending             int    `json:"pending"`
	Failed              int    `json:"failed"`
	Dead                int    `json:"dead"`
}

type Integration struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Type      string             `json:"type"`
	Enabled   bool               `json:"enabled"`
	Config    json.RawMessage    `json:"config"`
	Filters   IntegrationFilters `json:"filters"`
	Health    IntegrationHealth  `json:"health"`
	CreatedAt string             `json:"created_at"`
	UpdatedAt string             `json:"updated_at"`
}

type Delivery struct {
	IntegrationID     string `json:"integration_id"`
	FindingKey        string `json:"finding_key"`
	LastAction        string `json:"last_action"`
	State             string `json:"state"`
	Attempts          int    `json:"attempts"`
	NextAttemptAt     string `json:"next_attempt_at"`
	LastError         string `json:"last_error"`
	SentAt            string `json:"sent_at"`
	OccurrencesAtSend int    `json:"occurrences_at_send"`
	SeverityAtSend    string `json:"severity_at_send"`
	StatusAtSend      string `json:"status_at_send"`
	ExternalID        string `json:"external_id"`
	UpdatedAt         string `json:"updated_at"`
}

type DeliveryAttempt struct {
	IntegrationID string `json:"integration_id"`
	FindingKey    string `json:"finding_key"`
	Action        string `json:"action"`
	Attempt       int    `json:"attempt"`
	Outcome       string `json:"outcome"`
	HTTPStatus    int    `json:"http_status"`
	Error         string `json:"error"`
	DurationMs    int    `json:"duration_ms"`
	EventCount    int    `json:"event_count"`
	AttemptedAt   string `json:"attempted_at"`
}

type IntegrationWrite struct {
	Name    string             `json:"name"`
	Type    string             `json:"type"`
	Enabled bool               `json:"enabled"`
	Config  json.RawMessage    `json:"config"`
	Filters IntegrationFilters `json:"filters"`
}
