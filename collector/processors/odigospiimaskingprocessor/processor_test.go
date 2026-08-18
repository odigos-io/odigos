package odigospiimaskingprocessor

import (
	"context"
	"fmt"
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

// keyByWorkloadExtension resolves the cache key from a resource attribute, so a single batch can
// carry spans from several workloads with different masking configs.
type keyByWorkloadExtension struct {
	stubOdigosConfigExtension
	unregistered bool
}

func (e *keyByWorkloadExtension) GetWorkloadCacheKey(res pcommon.Resource) (string, error) {
	workload, ok := res.Attributes().Get("k8s.container.name")
	if !ok {
		return "", fmt.Errorf("resource carries no workload identity")
	}
	return workload.Str(), nil
}

func (e *keyByWorkloadExtension) UnregisterWorkloadConfigCacheCallback(collector.WorkloadConfigCacheCallback) {
	e.unregistered = true
}

// generateWorkloadTrace builds one resource span per workload, each with a single span carrying the
// given message attribute. A workload named "" gets no identity attribute at all.
func generateWorkloadTrace(messagesByWorkload map[string]string) ptrace.Traces {
	traces := ptrace.NewTraces()
	for workload, message := range messagesByWorkload {
		rs := traces.ResourceSpans().AppendEmpty()
		if workload != "" {
			rs.Resource().Attributes().PutStr("k8s.container.name", workload)
		}
		span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		span.SetName("test")
		span.Attributes().PutStr("message", message)
	}
	return traces
}

func messageForWorkload(t *testing.T, traces ptrace.Traces, workload string) string {
	t.Helper()
	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		name, ok := rs.Resource().Attributes().Get("k8s.container.name")
		if workload == "" {
			if ok {
				continue
			}
		} else if !ok || name.Str() != workload {
			continue
		}
		message, ok := rs.ScopeSpans().At(0).Spans().At(0).Attributes().Get("message")
		require.True(t, ok, "span of workload %q has no message attribute", workload)
		return message.Str()
	}
	t.Fatalf("no resource span found for workload %q", workload)
	return ""
}

func emailMaskingConfig() *commonapi.ContainerCollectorConfig {
	return &commonapi.ContainerCollectorConfig{
		PiiMasking: &actions.PiiMaskingConfig{PiiCategories: []actions.PiiCategory{actions.EmailMasking}},
	}
}

// The masking rules of a workload live in a cache the config extension updates at runtime. A
// workload whose config is removed, replaced or rejected must not affect any other workload, since
// that would silently stop masking traffic the user did configure.
func TestOnSetUpdatesOnlyItsOwnWorkload(t *testing.T) {
	const message = "contact user@example.com"

	tests := []struct {
		name       string
		update     *commonapi.ContainerCollectorConfig
		wantMasked string
		// Workloads with nothing to mask are dropped from the cache instead of being kept with an
		// empty rule set, which would walk every attribute of every span for nothing.
		wantCached bool
	}{
		{
			name:       "a replaced config applies the new rules instead of the old ones",
			update:     &commonapi.ContainerCollectorConfig{PiiMasking: &actions.PiiMaskingConfig{PiiCategories: []actions.PiiCategory{actions.UuidMasking}}},
			wantMasked: message,
			wantCached: true,
		},
		{
			name:       "removing the masking config stops masking",
			update:     &commonapi.ContainerCollectorConfig{},
			wantMasked: message,
		},
		{
			name: "a config with nothing to mask stops masking",
			update: &commonapi.ContainerCollectorConfig{
				PiiMasking: &actions.PiiMaskingConfig{},
			},
			wantMasked: message,
		},
		{
			// Rejected configs leave the workload unmasked rather than keeping the previous rules.
			name: "an unsupported category is rejected as a whole",
			update: &commonapi.ContainerCollectorConfig{
				PiiMasking: &actions.PiiMaskingConfig{
					PiiCategories: []actions.PiiCategory{actions.EmailMasking, "not-a-category"},
				},
			},
			wantMasked: message,
		},
		{
			name: "a custom regex without a capture group is rejected as a whole",
			update: &commonapi.ContainerCollectorConfig{
				PiiMasking: &actions.PiiMaskingConfig{
					PiiCategories:       []actions.PiiCategory{actions.EmailMasking},
					CustomRegexMaskings: []actions.CustomRegexMasking{{Regex: "secret"}},
				},
			},
			wantMasked: message,
		},
		{
			name:       "a still valid config keeps masking",
			update:     emailMaskingConfig(),
			wantMasked: "contact ***EMAIL***",
			wantCached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})
			proc.provider = &keyByWorkloadExtension{}
			proc.OnSet("updated", emailMaskingConfig())
			proc.OnSet("untouched", emailMaskingConfig())

			proc.OnSet("updated", tt.update)

			_, cached := proc.maskersCache.get("updated")
			assert.Equal(t, tt.wantCached, cached, "presence of compiled rules for the updated workload")

			out, err := proc.processTraces(context.Background(), generateWorkloadTrace(map[string]string{
				"updated":   message,
				"untouched": message,
			}))
			require.NoError(t, err)

			assert.Equal(t, tt.wantMasked, messageForWorkload(t, out, "updated"))
			assert.Equal(t, "contact ***EMAIL***", messageForWorkload(t, out, "untouched"),
				"the config of one workload must not affect another")
		})
	}
}

func TestOnDeleteKeyRemovesOnlyItsOwnWorkload(t *testing.T) {
	const message = "contact user@example.com"

	proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})
	proc.provider = &keyByWorkloadExtension{}
	proc.OnSet("deleted", emailMaskingConfig())
	proc.OnSet("kept", emailMaskingConfig())

	proc.OnDeleteKey("deleted")

	out, err := proc.processTraces(context.Background(), generateWorkloadTrace(map[string]string{
		"deleted": message,
		"kept":    message,
		// A resource the extension cannot identify is passed through untouched.
		"": message,
	}))
	require.NoError(t, err)

	assert.Equal(t, message, messageForWorkload(t, out, "deleted"))
	assert.Equal(t, "contact ***EMAIL***", messageForWorkload(t, out, "kept"))
	assert.Equal(t, message, messageForWorkload(t, out, ""))
}

func TestShutdownStopsMaskingAndReleasesTheCallback(t *testing.T) {
	proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})
	ext := &keyByWorkloadExtension{}
	proc.provider = ext
	proc.OnSet("app", emailMaskingConfig())

	require.NoError(t, proc.Shutdown(context.Background()))

	// A registration that outlives the processor keeps the extension calling into it.
	assert.True(t, ext.unregistered, "the workload config callback must be unregistered")
	if _, cached := proc.maskersCache.get("app"); cached {
		t.Error("compiled masking rules must not survive shutdown")
	}

	out, err := proc.processTraces(context.Background(), generateWorkloadTrace(map[string]string{
		"app": "contact user@example.com",
	}))
	require.NoError(t, err)
	assert.Equal(t, "contact user@example.com", messageForWorkload(t, out, "app"))
}

// registeringExtension records the callbacks registered with it and doubles as a collector
// component so it can be handed to the processor through a host.
type registeringExtension struct {
	keyByWorkloadExtension
	registered  []collector.WorkloadConfigCacheCallback
	cacheSynced bool
}

func (e *registeringExtension) Start(context.Context, component.Host) error { return nil }

func (e *registeringExtension) Shutdown(context.Context) error { return nil }

func (e *registeringExtension) RegisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	e.registered = append(e.registered, cb)
}

func (e *registeringExtension) WaitForCacheSync(context.Context) bool { return e.cacheSynced }

// plainExtension is a collector component that is not an OdigosConfigExtension.
type plainExtension struct{}

func (plainExtension) Start(context.Context, component.Host) error { return nil }

func (plainExtension) Shutdown(context.Context) error { return nil }

type stubHost struct {
	extensions map[component.ID]component.Component
}

func (h stubHost) GetExtensions() map[component.ID]component.Component { return h.extensions }

// Start is what connects the processor to the per-workload config it masks by. Failing to register
// the callback leaves the masking cache empty forever, which looks exactly like "no PII rules
// configured" instead of an error.
func TestStartResolvesTheConfigExtension(t *testing.T) {
	extID := component.MustNewID("odigosconfigk8s")
	otherID := component.MustNewID("odigosconfigvm")

	t.Run("registers the processor for workload config updates", func(t *testing.T) {
		proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{OdigosConfigExtension: &extID})
		ext := &registeringExtension{cacheSynced: true}

		require.NoError(t, proc.Start(context.Background(), stubHost{
			extensions: map[component.ID]component.Component{extID: ext},
		}))

		assert.Same(t, ext, proc.provider)
		require.Len(t, ext.registered, 1)
		assert.Same(t, proc, ext.registered[0])
	})

	t.Run("an incomplete cache sync is not fatal", func(t *testing.T) {
		// Spans that arrive before the sync completes go out unmasked, but refusing to start would
		// take down the whole collector instead.
		proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{OdigosConfigExtension: &extID})
		ext := &registeringExtension{cacheSynced: false}

		require.NoError(t, proc.Start(context.Background(), stubHost{
			extensions: map[component.ID]component.Component{extID: ext},
		}))
		assert.Same(t, ext, proc.provider)
	})

	errorCases := []struct {
		name    string
		cfg     *Config
		host    stubHost
		wantErr string
	}{
		{
			name:    "no extension configured",
			cfg:     &Config{},
			wantErr: "odigos_config_extension is required",
		},
		{
			name:    "the configured extension is not enabled",
			cfg:     &Config{OdigosConfigExtension: &extID},
			host:    stubHost{extensions: map[component.ID]component.Component{otherID: &registeringExtension{}}},
			wantErr: "not found",
		},
		{
			name:    "the configured extension is of another kind",
			cfg:     &Config{OdigosConfigExtension: &extID},
			host:    stubHost{extensions: map[component.ID]component.Component{extID: plainExtension{}}},
			wantErr: "is not an OdigosConfigExtension",
		},
	}

	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), tt.cfg)

			err := proc.Start(context.Background(), tt.host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Nil(t, proc.provider, "a failed start must not leave a provider behind")
		})
	}
}

// Header and similar attributes reach the collector as arrays, so PII in them is only masked if the
// slice is walked. A string-only implementation leaks them without failing anywhere.
func TestExtension_MasksPiiInsideSliceAttributes(t *testing.T) {
	proc := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{})
	ext := &stubOdigosConfigExtension{key: "default/deployment/app/container"}
	proc.provider = ext
	proc.OnSet(ext.key, emailMaskingConfig())

	traces := ptrace.NewTraces()
	span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	header := span.Attributes().PutEmptySlice("http.request.header.from")
	header.AppendEmpty().SetStr("user@example.com")
	header.AppendEmpty().SetStr("no pii here")
	header.AppendEmpty().SetInt(7)
	nested := header.AppendEmpty().SetEmptySlice()
	nested.AppendEmpty().SetStr("nested user@example.com")

	out, err := proc.processTraces(context.Background(), traces)
	require.NoError(t, err)

	got := out.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes()
	masked, ok := got.Get("http.request.header.from")
	require.True(t, ok)
	values := masked.Slice()
	require.Equal(t, 4, values.Len())
	assert.Equal(t, "***EMAIL***", values.At(0).Str())
	assert.Equal(t, "no pii here", values.At(1).Str())
	assert.Equal(t, int64(7), values.At(2).Int())
	assert.Equal(t, "nested ***EMAIL***", values.At(3).Slice().At(0).Str())
}
