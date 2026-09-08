package insights

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertPolicyAndReadReturnsPersistedPolicy(t *testing.T) {
	var upserted Policy
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&upserted))
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			// The engine assigns the id and defaults fire_at_score.
			_, _ = w.Write([]byte(`{"count":1,"items":[{
				"id": 991,
				"name": "checkout",
				"enabled": true,
				"fire_at_score": 60,
				"scope": "service",
				"scope_key": "prod/checkout"
			}]}`))
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	stored, err := client.UpsertPolicyAndRead(context.Background(), Policy{
		Name:     "checkout",
		Enabled:  true,
		Scope:    "service",
		ScopeKey: "prod/checkout",
	})
	require.NoError(t, err)
	assert.Equal(t, "checkout", upserted.Name)
	assert.Equal(t, int64(991), stored.ID)
	assert.Equal(t, 60, stored.FireAtScore)
}

func TestUpsertPolicyAndReadWhenAbsentFromReadBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"count":0,"items":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	// Listing skips disabled policies, so their absence is expected and the
	// accepted request is echoed back.
	disabled := Policy{Name: "checkout", Scope: "service", ScopeKey: "prod/checkout"}
	stored, err := client.UpsertPolicyAndRead(context.Background(), disabled)
	require.NoError(t, err)
	assert.Equal(t, disabled, stored)

	// An enabled policy that is missing after the write is a real failure.
	enabled := disabled
	enabled.Enabled = true
	_, err = client.UpsertPolicyAndRead(context.Background(), enabled)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInternal)
}

func TestUpsertLearningPolicyAndReadMatchesOnClassAndScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"count":2,"items":[
			{"class":"D2_egress","mode":"all","scope":"global","scope_key":""},
			{"class":"D3_latency","mode":"any","min_matches":2,"scope":"service","scope_key":"prod/checkout"}
		]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	stored, err := client.UpsertLearningPolicyAndRead(context.Background(), LearningPolicy{
		Class:    "D3_latency",
		Mode:     "any",
		Scope:    "service",
		ScopeKey: "prod/checkout",
	})
	require.NoError(t, err)
	require.NotNil(t, stored.MinMatches)
	assert.Equal(t, 2, *stored.MinMatches)

	_, err = client.UpsertLearningPolicyAndRead(context.Background(), LearningPolicy{
		Class:    "D8_payload_size",
		Mode:     "all",
		Scope:    "global",
		ScopeKey: "",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInternal)
}

func TestUpsertGuardrailAndReadReturnsPersistedRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"count":1,"items":[{
			"scope": "service",
			"scope_key": "prod/checkout",
			"rules": [{"key":"allowed_egress","label":"Allowed egress","mode":"enforce","allowlist":["db:5432"]}]
		}]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	stored, err := client.UpsertGuardrailAndRead(context.Background(), Guardrail{
		Scope:    "service",
		ScopeKey: "prod/checkout",
		Rules:    []GuardrailRule{{Key: "allowed_egress", Mode: "enforce"}},
	})
	require.NoError(t, err)
	require.Len(t, stored.Rules, 1)
	assert.Equal(t, "Allowed egress", stored.Rules[0].Label)
	assert.Equal(t, []string{"db:5432"}, stored.Rules[0].Allowlist)
}

func TestUpdateSystemSettingsAndReadReturnsServerDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{
			"sampling": {"examples_per_transaction": 3, "example_sample_interval_seconds": 60},
			"retention": {"observation_retention_days": 14},
			"findings": {"default_window_hours": 24, "max_window_hours": 168},
			"capacity": {"max_resident_transactions": 1000, "max_baseline_set_members": 500},
			"writeback": {"flush_interval_seconds": 30},
			"detection": {"auto_transaction_guardrail": true},
			"identity": {"transaction_identity_dimensions": [{"key": "http.response.status_code", "enabled": true}]}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	stored, err := client.UpdateSystemSettingsAndRead(context.Background(), SystemSettings{
		Sampling: SystemSamplingSettings{ExamplesPerTransaction: 3},
	})
	require.NoError(t, err)
	assert.Equal(t, 168, stored.Findings.MaxWindowHours)
	assert.Equal(t, 30, stored.Writeback.FlushIntervalSeconds)
	require.Len(t, stored.Identity.TransactionIdentityDimensions, 1)
	assert.Equal(t, "http.response.status_code", stored.Identity.TransactionIdentityDimensions[0].Key)
	assert.True(t, stored.Identity.TransactionIdentityDimensions[0].Enabled)
}

func TestUpsertPolicyAndReadPropagatesWriteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method, "read-back must not run after a failed write")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":"unavailable","message":"engine starting"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	_, err = client.UpsertPolicyAndRead(context.Background(), Policy{Scope: "global"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEngineStarting)
}

// A write that succeeds followed by a failed or stale read-back must not look
// like a successful save: the UI would show the pre-edit entity as persisted.
func TestUpsertAndReadPropagatesReadBackFailures(t *testing.T) {
	tests := []struct {
		name         string
		listResponse string
		listStatus   int
		call         func(*Client) error
		wantErr      error
	}{
		{
			name:       "policy read-back fails",
			listStatus: http.StatusInternalServerError,
			call: func(c *Client) error {
				_, err := c.UpsertPolicyAndRead(context.Background(), Policy{Enabled: true})
				return err
			},
			wantErr: ErrInternal,
		},
		{
			name:         "learning policy is missing after the upsert",
			listResponse: `{"count":0,"items":[]}`,
			call: func(c *Client) error {
				_, err := c.UpsertLearningPolicyAndRead(context.Background(), LearningPolicy{Class: "D2_egress", Scope: "global"})
				return err
			},
			wantErr: ErrInternal,
		},
		{
			name:       "learning policy read-back fails",
			listStatus: http.StatusInternalServerError,
			call: func(c *Client) error {
				_, err := c.UpsertLearningPolicyAndRead(context.Background(), LearningPolicy{Class: "D2_egress"})
				return err
			},
			wantErr: ErrInternal,
		},
		{
			name:         "guardrail is missing after the upsert",
			listResponse: `{"count":0,"items":[]}`,
			call: func(c *Client) error {
				_, err := c.UpsertGuardrailAndRead(context.Background(), Guardrail{ScopeKey: "prod/checkout"})
				return err
			},
			wantErr: ErrInternal,
		},
		{
			name:       "guardrail read-back fails",
			listStatus: http.StatusInternalServerError,
			call: func(c *Client) error {
				_, err := c.UpsertGuardrailAndRead(context.Background(), Guardrail{ScopeKey: "prod/checkout"})
				return err
			},
			wantErr: ErrInternal,
		},
		{
			name:       "system settings read-back fails",
			listStatus: http.StatusInternalServerError,
			call: func(c *Client) error {
				_, err := c.UpdateSystemSettingsAndRead(context.Background(), SystemSettings{})
				return err
			},
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				if tt.listStatus != 0 {
					w.WriteHeader(tt.listStatus)
					_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"engine failed"}}`))
					return
				}
				_, _ = w.Write([]byte(tt.listResponse))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			err = tt.call(client)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// A disabled policy is allowed to be absent from the read-back, because the
// engine drops it; echoing the request keeps the UI from reporting a failure.
func TestUpsertPolicyAndReadEchoesADisabledPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"count":0,"items":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	requested := Policy{Name: "checkout", Enabled: false, Scope: "service", ScopeKey: "prod/checkout"}
	stored, err := client.UpsertPolicyAndRead(context.Background(), requested)
	require.NoError(t, err)
	assert.Equal(t, requested, stored)
}

// The read-back has to match on the full key. Matching only part of it would
// return a sibling entity as the just-saved one.
func TestUpsertAndReadMatchOnTheFullKey(t *testing.T) {
	t.Run("policy matches scope and scope key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_, _ = w.Write([]byte(`{"count":2,"items":[
				{"id":1,"scope":"global","scope_key":"prod/checkout"},
				{"id":2,"scope":"service","scope_key":"prod/payments"},
				{"id":3,"scope":"service","scope_key":"prod/checkout"}
			]}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL)
		require.NoError(t, err)

		stored, err := client.UpsertPolicyAndRead(context.Background(), Policy{Enabled: true, Scope: "service", ScopeKey: "prod/checkout"})
		require.NoError(t, err)
		assert.Equal(t, int64(3), stored.ID)
	})

	t.Run("learning policy matches class, scope and scope key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_, _ = w.Write([]byte(`{"count":3,"items":[
				{"class":"D3_latency","mode":"a","scope":"service","scope_key":"prod/checkout"},
				{"class":"D2_egress","mode":"b","scope":"global","scope_key":"prod/checkout"},
				{"class":"D2_egress","mode":"c","scope":"service","scope_key":"prod/payments"},
				{"class":"D2_egress","mode":"d","scope":"service","scope_key":"prod/checkout"}
			]}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL)
		require.NoError(t, err)

		stored, err := client.UpsertLearningPolicyAndRead(context.Background(), LearningPolicy{
			Class:    "D2_egress",
			Scope:    "service",
			ScopeKey: "prod/checkout",
		})
		require.NoError(t, err)
		assert.Equal(t, LearningMode("d"), stored.Mode)
	})
}
