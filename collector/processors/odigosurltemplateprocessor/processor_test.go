package odigosurltemplateprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func addSpanToTraces(traces ptrace.Traces, serviceName, spanName string, kind ptrace.SpanKind, spanAttrs map[string]any) {
	rs := traces.ResourceSpans().AppendEmpty()
	if serviceName != "" {
		rs.Resource().Attributes().PutStr(string(semconv.ServiceNameKey), serviceName)
	}
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(spanAttrs)
	span.SetName(spanName)
	span.SetKind(kind)
}

func generateTraceData(serviceName, spanName string, kind ptrace.SpanKind, spanAttrs map[string]any) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	if serviceName != "" {
		rs.Resource().Attributes().PutStr(string(semconv.ServiceNameKey), serviceName)
	}
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(spanAttrs)
	span.SetName(spanName)
	span.SetKind(kind)
	return td
}

func assertSpanNameAndAttribute(t *testing.T, span ptrace.Span, expectedName string, expectedAttrKey string, expectedAttrValue any) {
	require.Equal(t, expectedName, span.Name())
	attrValue, found := span.Attributes().Get(expectedAttrKey)
	if expectedAttrValue == "" {
		require.False(t, found)
	} else {
		require.True(t, found)
		require.Equal(t, expectedAttrValue, attrValue.AsString())
	}
}

func TestProcessor_Traces(t *testing.T) {
	set := processortest.NewNopSettings(processortest.NopType)

	processor, err := newUrlTemplateProcessor(set, &Config{})
	require.NoError(t, err)

	tt := []struct {
		name              string
		serviceName       string
		spanKind          ptrace.SpanKind
		inputSpanName     string
		inputSpanAttrs    map[string]any
		expectedSpanName  string
		expectedAttrKey   string
		expectedAttrValue string
	}{
		{
			name:          "numeric id in url path",
			serviceName:   "numeric-templated-string",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/1234",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "uuid in url path",
			serviceName:   "uuid-templated-string",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/123e4567-e89b-12d3-a456-426614174000",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "uuid with any suffix",
			serviceName:   "uuid-templated-string-with-suffix",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/processes/123e4567-e89b-12d3-a456-426614174000_PROCESS",
			},
			expectedSpanName:  "GET /processes/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/processes/{id}",
		},
		{
			name:          "uuid with any prefix",
			serviceName:   "uuid-with-any-prefix",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/processes/PROCESS_123e4567-e89b-12d3-a456-426614174000",
			},
			expectedSpanName:  "GET /processes/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/processes/{id}",
		},
		{
			name:          "multiple numeric ids in url path",
			serviceName:   "multiple-numeric-templated-string",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/1234/friends/4567",
			},
			expectedSpanName:  "GET /user/{id}/friends/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}/friends/{id}",
		},
		{
			name:          "deprecated method attribute",
			serviceName:   "deprecated-method-attribute",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.method": "GET",
				"url.path":    "/user/1234",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "deprecated http.target attribute",
			serviceName:   "deprecated-http-target-attribute",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"http.target":         "/user/1234",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "http.target with query params",
			serviceName:   "http-target-with-query-params",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"http.target":         "/user/1234?name=John",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "with url.full attribute",
			serviceName:   "url-full-attribute",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.full":            "http://example.com/user/1234?name=John",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "with deprecated http.url attribute",
			serviceName:   "http-url-attribute",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"http.url":            "http://example.com/user/1234?name=John",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "client span",
			serviceName:   "client-span",
			spanKind:      ptrace.SpanKindClient,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.full":            "http://example.com/user/1234?name=John",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "url.template",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "span name is not the method",
			serviceName:   "span-name-not-the-method",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "some-other-name",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/1234",
			},
			expectedSpanName:  "some-other-name", // should not be modified
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}", // should exist
		},
		{
			name:          "ignore internal span",
			serviceName:   "ignore-internal-span",
			spanKind:      ptrace.SpanKindInternal,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/1234",
			},
			expectedSpanName:  "GET", // should not be modified
			expectedAttrKey:   "http.route",
			expectedAttrValue: "", // should not exist
		},
		{
			name:          "ignore span without any path",
			serviceName:   "ignore-span-without-path",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
			},
			expectedSpanName:  "GET",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "", // should not exist
		},
		{
			name:          "ignore span without any method",
			serviceName:   "ignore-span-without-method",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"url.path": "/user/1234",
			},
			expectedSpanName:  "GET",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "", // should not exist
		},
		{
			name:          "ignore server span with templated attribute",
			serviceName:   "ignore-server-span-with-templated-attribute",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/1234",
				"http.route":          "/user/<id>",
			},
			expectedSpanName:  "GET", // do not modify span name as the attribute exists
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/<id>", // should not be modified
		},
		{
			name:          "ignore client span with templated attribute",
			serviceName:   "ignore-client-span-with-templated-attribute",
			spanKind:      ptrace.SpanKindClient,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.full":            "http://example.com/user/1234?name=John",
				"url.template":        "/user/<id>",
			},
			expectedSpanName:  "GET", // do not modify span name as the attribute exists
			expectedAttrKey:   "url.template",
			expectedAttrValue: "/user/<id>", // should not be modified
		},
		{
			name:          "static url path",
			serviceName:   "static-url-path",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/products",
			},
			expectedSpanName:  "GET /products",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/products",
		},
		{
			name:          "mixed-numbers-and-text",
			serviceName:   "mixed-numbers-and-text",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/api/v1",
			},
			expectedSpanName:  "GET /api/v1",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/api/v1",
		},
		{
			name:          "hexencoded id",
			serviceName:   "hexencoded-id",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/6f2a9cdeab34f01e",
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "long hexencoded id",
			serviceName:   "long-hexencoded-id",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/6f2a9cdeab34f01e1234567890abcdef", // 32 chars
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name:          "short looking like hexencoded id",
			serviceName:   "short-looking-like-hexencoded-id",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/feed12",
			},
			expectedSpanName:  "GET /user/feed12", // should not be templated as the string contains hex chars, but it's too short
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/feed12",
		},
		{
			name:          "long text",
			serviceName:   "long-text",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/CamelCaseLongTextThatShouldNotBeTemplated",
			},
			expectedSpanName:  "GET /user/CamelCaseLongTextThatShouldNotBeTemplated", // should not be templated as chars are not hex
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/CamelCaseLongTextThatShouldNotBeTemplated",
		},
		{
			name:          "non-even length hex",
			serviceName:   "non-even-length-hex",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/abcdefabcdefabcde", // contains 17 chars
			},
			expectedSpanName:  "GET /user/abcdefabcdefabcde", // should not be templated as the string contains hex chars, but it's too short
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/abcdefabcdefabcde",
		},
		{
			name:          "long number with text",
			serviceName:   "long-number-with-text",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/INC0012686", // contains 7 digits number
			},
			expectedSpanName:  "GET /user/{id}", // should be templated as the number is long
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name: "long number in middle of text",
			// this is a corner case where the number is long, but it is not at the beginning or end of the string
			serviceName:   "long-number-in-middle-of-text",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/INC0012637US", // contains 7 digits number
			},
			expectedSpanName:  "GET /user/{id}",
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/{id}",
		},
		{
			name: "6 digits number should not be templated",
			// this is a corner case where the number is under the limit of digits (7)
			serviceName:   "6-digits-number-should-not-be-templated",
			spanKind:      ptrace.SpanKindServer,
			inputSpanName: "GET",
			inputSpanAttrs: map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/123456_654321", // contains 6 digits number twice
			},
			expectedSpanName:  "GET /user/123456_654321", // should not be templated as the number is under the limit
			expectedAttrKey:   "http.route",
			expectedAttrValue: "/user/123456_654321",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			traces := generateTraceData(tc.serviceName, tc.inputSpanName, tc.spanKind, tc.inputSpanAttrs)

			ctx := context.Background()
			processedTraces, err := processor.processTraces(ctx, traces)
			require.NoError(t, err)

			processedSpan := processedTraces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
			assertSpanNameAndAttribute(t, processedSpan, tc.expectedSpanName, tc.expectedAttrKey, tc.expectedAttrValue)
		})
	}
}

func TestProcessor_TemplatizationRules(t *testing.T) {
	tt := []struct {
		name              string
		rules             []string
		path              string
		expectedName      string
		expectedHttpRoute string
	}{
		{
			name:              "simple-templatization",
			rules:             []string{"/user/{user-name}"},
			path:              "/user/john",
			expectedName:      "GET /user/{user-name}",
			expectedHttpRoute: "/user/{user-name}",
		},
		{
			name:              "multiple-templatization",
			rules:             []string{"/user/{user-id}/friends/{friend-id}"},
			path:              "/user/1234/friends/4567",
			expectedName:      "GET /user/{user-id}/friends/{friend-id}",
			expectedHttpRoute: "/user/{user-id}/friends/{friend-id}",
		},
		{
			name:              "regex-templatization",
			rules:             []string{"/user/{user-id:\\d+}"},
			path:              "/user/1234",
			expectedName:      "GET /user/{user-id}",
			expectedHttpRoute: "/user/{user-id}",
		},
		{
			name:              "regex-templatization-fail",
			rules:             []string{"/user/{user-id:\\d+}"},
			path:              "/user/john",
			expectedName:      "GET /user/john",
			expectedHttpRoute: "/user/john",
		},
		{
			name:              "path-no-leading-slash",
			rules:             []string{"user/{user-id}"},
			path:              "user/1234",
			expectedName:      "GET user/{user-id}",
			expectedHttpRoute: "user/{user-id}",
		},
		{
			name:              "rule-overrides-default-templatization",
			rules:             []string{"/api/1"},
			path:              "/api/1",
			expectedName:      "GET /api/1", // 1 is not templatized because the rule specifies it
			expectedHttpRoute: "/api/1",
		},
		{
			name:              "exact-match",
			rules:             []string{"/user/{user-name}"},
			path:              "/user/john/children",
			expectedName:      "GET /user/john/children", // rule didn't match since we have 3 path segments and rule is for 2
			expectedHttpRoute: "/user/john/children",
		},
		{
			name:              "ignored-on-static-string-mismatch",
			rules:             []string{"/user/{user-name}"},
			path:              "/product/spoon",
			expectedName:      "GET /product/spoon", // rule is on user and path contains product
			expectedHttpRoute: "/product/spoon",
		},
		{
			name: "multi-matching-rules",
			rules: []string{
				"/user/john",
				"/user/{user-name}",
			},
			path:              "/user/john",
			expectedName:      "GET /user/john", // john appears in the path before {user-name} so it is not templatized
			expectedHttpRoute: "/user/john",
		},
		{
			name: "second-rule-matches",
			rules: []string{
				"/user/john",
				"/user/{user-name}",
			},
			path:              "/user/jane",
			expectedName:      "GET /user/{user-name}",
			expectedHttpRoute: "/user/{user-name}",
		},
		{
			name:              "missing-section-name",
			rules:             []string{"/user/{}"},
			path:              "/user/john",
			expectedName:      "GET /user/{id}", // fallback to name "id" when missing
			expectedHttpRoute: "/user/{id}",
		},
		{
			name:              "regexp-no-name",
			rules:             []string{"/user/{:[0-9]+}"},
			path:              "/user/1234",
			expectedName:      "GET /user/{id}", // fallback to name "id" when missing
			expectedHttpRoute: "/user/{id}",
		},
		{
			name:              "segment-rule-with-spaces",
			rules:             []string{"/user/{user-name : [a-zA-Z]+}"},
			path:              "/user/John",
			expectedName:      "GET /user/{user-name}",
			expectedHttpRoute: "/user/{user-name}",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			spanAttr := map[string]any{
				"http.request.method": "GET",
				"url.path":            tc.path,
			}
			traces := generateTraceData("test-service-name", "GET", ptrace.SpanKindServer, spanAttr)
			// Add the templated rule to the processor
			processor, err := newUrlTemplateProcessor(processortest.NewNopSettings(processortest.NopType), &Config{TemplatizationRules: tc.rules})
			require.NoError(t, err)
			// Process the traces
			ctx := context.Background()
			processedTraces, err := processor.processTraces(ctx, traces)
			require.NoError(t, err)
			// Get the processed span
			processedSpan := processedTraces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
			// Assert the span name and http.route attribute
			assertSpanNameAndAttribute(t, processedSpan, tc.expectedName, "http.route", tc.expectedHttpRoute)
		})
	}
}

func TestProcessor_CustomIdsRegexp(t *testing.T) {
	tt := []struct {
		name              string
		customIds         []string
		path              string
		expectedName      string
		expectedHttpRoute string
	}{
		{
			name:              "custom id",
			customIds:         []string{"^in_[0-9]+$"},
			path:              "/product/in_005",
			expectedName:      "GET /product/{id}",
			expectedHttpRoute: "/product/{id}",
		},
		{
			name:              "multiple custom ids",
			customIds:         []string{"^in_[0-9]+$", "^out_[0-9]+$"},
			path:              "/foo/out_005/bar/in_123",
			expectedName:      "GET /foo/{id}/bar/{id}",
			expectedHttpRoute: "/foo/{id}/bar/{id}",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			spanAttr := map[string]any{
				"http.request.method": "GET",
				"url.path":            tc.path,
			}
			traces := generateTraceData("test-service-name", "GET", ptrace.SpanKindServer, spanAttr)
			// Add the templated rule to the processor
			processor, err := newUrlTemplateProcessor(processortest.NewNopSettings(processortest.NopType), &Config{CustomIdsRegexp: tc.customIds})
			require.NoError(t, err)
			// Process the traces
			ctx := context.Background()
			processedTraces, err := processor.processTraces(ctx, traces)
			require.NoError(t, err)
			// Get the processed span
			processedSpan := processedTraces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
			// Assert the span name and http.route attribute
			assertSpanNameAndAttribute(t, processedSpan, tc.expectedName, "http.route", tc.expectedHttpRoute)
		})
	}
}
