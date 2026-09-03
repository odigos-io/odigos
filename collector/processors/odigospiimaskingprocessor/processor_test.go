package odigospiimaskingprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/collector"
)

func TestMaskPiiData_CategoriesAndCustom(t *testing.T) {
	cfg, err := compilePiiMaskingConfig(&actions.PiiMaskingConfig{
		PiiCategories: []actions.PiiCategory{actions.EmailMasking},
		CustomFormatMaskings: []actions.CustomFormatMasking{
			{LookupKey: "ssn", DataFormat: actions.FormatJSON},
		},
		CustomRegexMaskings: []actions.CustomRegexMasking{
			{Regex: `api[_-]?key=([^\s&]+)`},
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "email category",
			input: "contact me at user@example.com please",
			want:  "contact me at ***EMAIL*** please",
		},
		{
			name:  "json format masking",
			input: `{"ssn": "123-45-6789", "name": "alice"}`,
			want:  `{"ssn": "****", "name": "alice"}`,
		},
		{
			name:  "custom regex masking",
			input: "auth api_key=super-secret-value next",
			want:  "auth api_key=**** next",
		},
		{
			name:  "combined",
			input: `email=user@example.com payload={"ssn":"999"} api-key=abc123`,
			want:  `email=***EMAIL*** payload={"ssn":"****"} api-key=****`,
		},
		{
			name:  "no match",
			input: "nothing sensitive here",
			want:  "nothing sensitive here",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := maskPiiData(tc.input, cfg)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.input != tc.want, changed)
		})
	}
}

func TestBuildFormatMaskingRegexes(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		format      actions.DataFormat
		input       string
		want        string
		wantChanged bool
	}{
		{
			name:        "json",
			key:         "user_id",
			format:      actions.FormatJSON,
			input:       `{"user_id": "abc123", "name": "foo"}`,
			want:        `{"user_id": "****", "name": "foo"}`,
			wantChanged: true,
		},
		{
			name:        "json quoted value with spaces is masked in full",
			key:         "full_name",
			format:      actions.FormatJSON,
			input:       `{"full_name": "Jane Q Public", "id": 7}`,
			want:        `{"full_name": "****", "id": 7}`,
			wantChanged: true,
		},
		{
			name:        "json quoted value with comma is masked in full",
			key:         "full_name",
			format:      actions.FormatJSON,
			input:       `{"full_name": "Public, Jane"}`,
			want:        `{"full_name": "****"}`,
			wantChanged: true,
		},
		{
			name:        "json quoted value with escaped quotes is masked in full",
			key:         "note",
			format:      actions.FormatJSON,
			input:       `{"note": "say \"hi\" now", "x": 1}`,
			want:        `{"note": "****", "x": 1}`,
			wantChanged: true,
		},
		{
			name:        "json unterminated quoted value is masked in full",
			key:         "address",
			format:      actions.FormatJSON,
			input:       `{"address": "742 Evergreen Terrace`,
			want:        `{"address": "****`,
			wantChanged: true,
		},
		{
			name:        "json unquoted value",
			key:         "user_id",
			format:      actions.FormatJSON,
			input:       `{user_id: 42, name: "foo"}`,
			want:        `{user_id: ****, name: "foo"}`,
			wantChanged: true,
		},
		{
			name:   "json empty value is left as is",
			key:    "ssn",
			format: actions.FormatJSON,
			input:  `{"ssn": ""}`,
			want:   `{"ssn": ""}`,
		},
		{
			name:   "json key substring does not match",
			key:    "user_id",
			format: actions.FormatJSON,
			input:  `{"my_user_id": "Jane Q Public"}`,
			want:   `{"my_user_id": "Jane Q Public"}`,
		},
		{
			name:        "sql",
			key:         "password",
			format:      actions.FormatSQL,
			input:       `WHERE password = 'hunter2' AND status = 'ok'`,
			want:        `WHERE password = '****' AND status = 'ok'`,
			wantChanged: true,
		},
		{
			name:        "sql quoted value with spaces is masked in full",
			key:         "password",
			format:      actions.FormatSQL,
			input:       `WHERE password = 'hunter2 is secret' AND status = 'ok'`,
			want:        `WHERE password = '****' AND status = 'ok'`,
			wantChanged: true,
		},
		{
			name:        "sql quoted value with comma is masked in full",
			key:         "address",
			format:      actions.FormatSQL,
			input:       `UPDATE users SET address = '742 Evergreen Terrace, Springfield' WHERE id = 1`,
			want:        `UPDATE users SET address = '****' WHERE id = 1`,
			wantChanged: true,
		},
		{
			name:        "sql unquoted value does not swallow the rest of the statement",
			key:         "user_id",
			format:      actions.FormatSQL,
			input:       `WHERE user_id = 42 AND status = 'ok'`,
			want:        `WHERE user_id = **** AND status = 'ok'`,
			wantChanged: true,
		},
		{
			name:        "sql tight quoted value",
			key:         "user_id",
			format:      actions.FormatSQL,
			input:       `WHERE user_id='abc'`,
			want:        `WHERE user_id='****'`,
			wantChanged: true,
		},
		{
			name:        "resource_path",
			key:         "orders",
			format:      actions.FormatResourcePath,
			input:       `/api/v1/orders/abc-123/items`,
			want:        `/api/v1/orders/****/items`,
			wantChanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := buildFormatMaskingRegexes(tc.key, tc.format)
			require.NoError(t, err)
			require.NotEmpty(t, res)

			got := tc.input
			changed := false
			for _, re := range res {
				masked, applied := maskCaptureGroups(re, got)
				if applied {
					got = masked
					changed = true
				}
			}
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantChanged, changed)
		})
	}
}

func TestMaskPiiData_FormatMaskingIsIdempotent(t *testing.T) {
	cfg, err := compilePiiMaskingConfig(&actions.PiiMaskingConfig{
		CustomFormatMaskings: []actions.CustomFormatMasking{
			{LookupKey: "full_name", DataFormat: actions.FormatJSON},
			{LookupKey: "password", DataFormat: actions.FormatSQL},
		},
	})
	require.NoError(t, err)

	first, changed := maskPiiData(`{"full_name": "Jane Q Public"} WHERE password = 'hunter2 is secret'`, cfg)
	require.True(t, changed)
	assert.Equal(t, `{"full_name": "****"} WHERE password = '****'`, first)

	second, _ := maskPiiData(first, cfg)
	assert.Equal(t, first, second)
}

func TestCompilePiiMaskingConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     actions.PiiMaskingConfig
		wantErr string
	}{
		{
			name: "valid",
			cfg: actions.PiiMaskingConfig{
				PiiCategories: []actions.PiiCategory{actions.EmailMasking},
				CustomFormatMaskings: []actions.CustomFormatMasking{
					{LookupKey: "ssn", DataFormat: actions.FormatJSON},
				},
				CustomRegexMaskings: []actions.CustomRegexMasking{
					{Regex: `(secret)`},
				},
			},
		},
		{
			name: "invalid category",
			cfg: actions.PiiMaskingConfig{
				PiiCategories: []actions.PiiCategory{"PHONE"},
			},
			wantErr: "unsupported category",
		},
		{
			name: "format missing lookupKey",
			cfg: actions.PiiMaskingConfig{
				CustomFormatMaskings: []actions.CustomFormatMasking{
					{DataFormat: actions.FormatJSON},
				},
			},
			wantErr: "lookupKey is required",
		},
		{
			name: "regex without capture group",
			cfg: actions.PiiMaskingConfig{
				CustomRegexMaskings: []actions.CustomRegexMasking{
					{Regex: `abc`},
				},
			},
			wantErr: "capture group",
		},
		{
			name: "invalid regex",
			cfg: actions.PiiMaskingConfig{
				CustomRegexMaskings: []actions.CustomRegexMasking{
					{Regex: `(`},
				},
			},
			wantErr: "invalid regex",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compilePiiMaskingConfig(&tc.cfg)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	extID := component.MustNewID("odigosconfigk8s")

	err := Config{}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "odigos_config_extension is required")

	err = Config{OdigosConfigExtension: &extID}.Validate()
	assert.NoError(t, err)
}

type stubOdigosConfigExtension struct {
	key string
	cfg *commonapi.ContainerCollectorConfig
}

func (s *stubOdigosConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	if s.cfg == nil {
		return nil, false
	}
	return s.cfg, true
}

func (s *stubOdigosConfigExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (s *stubOdigosConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return s.key, nil
}

func (s *stubOdigosConfigExtension) GetWorkloadIdentityFromResource(pcommon.Resource) (string, pcommon.Map, error) {
	return s.key, pcommon.NewMap(), nil
}

func (s *stubOdigosConfigExtension) RegisterWorkloadConfigCacheCallback(collector.WorkloadConfigCacheCallback) {
}

func (s *stubOdigosConfigExtension) UnregisterWorkloadConfigCacheCallback(collector.WorkloadConfigCacheCallback) {
}

func (s *stubOdigosConfigExtension) WaitForCacheSync(context.Context) bool { return true }

func (s *stubOdigosConfigExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return nil, false
}

func generateTestTrace(attrs map[string]string) ptrace.Traces {
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("test")
	for k, v := range attrs {
		span.Attributes().PutStr(k, v)
	}
	return traces
}

func TestExtension_PerSourceConfig(t *testing.T) {
	proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})

	ext := &stubOdigosConfigExtension{key: "default/deployment/app/container"}
	proc.provider = ext
	proc.OnSet(ext.key, &commonapi.ContainerCollectorConfig{
		PiiMasking: &actions.PiiMaskingConfig{
			PiiCategories: []actions.PiiCategory{actions.EmailMasking},
		},
	})

	traces := generateTestTrace(map[string]string{
		"message": "contact user@example.com",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	msg, ok := span.Attributes().Get("message")
	require.True(t, ok)
	require.Equal(t, "contact ***EMAIL***", msg.Str())
}

func TestExtension_CustomFormatMaskingMasksWholeValue(t *testing.T) {
	proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})

	ext := &stubOdigosConfigExtension{key: "default/deployment/app/container"}
	proc.provider = ext
	proc.OnSet(ext.key, &commonapi.ContainerCollectorConfig{
		PiiMasking: &actions.PiiMaskingConfig{
			CustomFormatMaskings: []actions.CustomFormatMasking{
				{LookupKey: "full_name", DataFormat: actions.FormatJSON},
				{LookupKey: "address", DataFormat: actions.FormatSQL},
			},
		},
	})

	traces := generateTestTrace(map[string]string{
		"http.request.body": `{"full_name": "Jane Q Public", "id": 7}`,
		"db.query.text":     `UPDATE users SET address = '742 Evergreen Terrace' WHERE id = 1`,
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)

	body, ok := span.Attributes().Get("http.request.body")
	require.True(t, ok)
	assert.Equal(t, `{"full_name": "****", "id": 7}`, body.Str())

	query, ok := span.Attributes().Get("db.query.text")
	require.True(t, ok)
	assert.Equal(t, `UPDATE users SET address = '****' WHERE id = 1`, query.Str())
}

func TestExtension_SkipsWhenNoConfig(t *testing.T) {
	proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})

	ext := &stubOdigosConfigExtension{key: "default/deployment/app/container"}
	proc.provider = ext

	traces := generateTestTrace(map[string]string{
		"message": "contact user@example.com",
	})

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	span := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	msg, ok := span.Attributes().Get("message")
	require.True(t, ok)
	require.Equal(t, "contact user@example.com", msg.Str())
}
