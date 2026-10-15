package insights

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// TraceSurge is an ended trace surge insights keeps. Record is the instrumentor's record of it: the
// trace surge as the Sampling status held it.
type TraceSurge struct {
	Name    string          `json:"name"`
	EndedAt time.Time       `json:"ended_at"`
	Record  json.RawMessage `json:"record"`
}

type TraceSurgeList struct {
	Items     []TraceSurge `json:"items"`
	Truncated bool         `json:"truncated"`
}

// ListTraceSurges returns the ended trace surges that overlap [from, to], the latest-ended first,
// at most limit, only those of ruleID when it is set.
func (c *Client) ListTraceSurges(ctx context.Context, from, to time.Time, ruleID string, limit int) (*TraceSurgeList, error) {
	endpoint := c.apiEndpoint("trace-surges")
	query := endpoint.Query()
	query.Set("from", from.UTC().Format(time.RFC3339))
	query.Set("to", to.UTC().Format(time.RFC3339))
	if ruleID != "" {
		query.Set("rule_id", ruleID)
	}
	query.Set("limit", strconv.Itoa(limit))
	endpoint.RawQuery = query.Encode()
	var result TraceSurgeList
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTraceSurge returns the ended trace surge named name, or an error that is ErrNotFound.
func (c *Client) GetTraceSurge(ctx context.Context, name string) (*TraceSurge, error) {
	var result TraceSurge
	if err := c.do(ctx, http.MethodGet, c.apiEndpoint("trace-surges", name), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
