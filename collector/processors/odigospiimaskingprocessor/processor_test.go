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

func TestBuildFormatMaskingRegex(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		format actions.DataFormat
		input  string
		want   string
	}{
		{
			name:   "json",
			key:    "user_id",
			format: actions.FormatJSON,
			input:  `{"user_id": "abc123", "name": "foo"}`,
			want:   `{"user_id": "****", "name": "foo"}`,
		},
		{
			name:   "sql",
			key:    "password",
			format: actions.FormatSQL,
			input:  `WHERE password = 'hunter2' AND status = 'ok'`,
			want:   `WHERE password = '****' AND status = 'ok'`,
		},
		{
			name:   "resource_path",
			key:    "orders",
			format: actions.FormatResourcePath,
			input:  `/api/v1/orders/abc-123/items`,
			want:   `/api/v1/orders/****/items`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re, err := buildFormatMaskingRegex(tc.key, tc.format)
			require.NoError(t, err)
			got, changed := maskCaptureGroups(re, tc.input)
			assert.True(t, changed)
			assert.Equal(t, tc.want, got)
		})
	}
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
