package odigospiimaskingprocessor

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestProcessTracesMasksEveryTraceAttributeContainer(t *testing.T) {
	traces := ptrace.NewTraces()
	resourceSpan := traces.ResourceSpans().AppendEmpty()
	resourceSpan.Resource().Attributes().PutStr("resource.email", "resource@example.com")
	scopeSpan := resourceSpan.ScopeSpans().AppendEmpty()
	scopeSpan.Scope().Attributes().PutStr("scope.email", "scope@example.com")
	span := scopeSpan.Spans().AppendEmpty()
	span.Attributes().PutStr("span.email", "span@example.com")
	span.Attributes().PutEmptySlice("span.email_list").AppendEmpty().SetStr("list@example.com")
	event := span.Events().AppendEmpty()
	event.Attributes().PutStr("event.email", "event@example.com")

	processor := newPiiMaskingProcessor(
		processortest.NewNopSettings(typ),
		&Config{
			PiiMaskingConfig: actions.PiiMaskingConfig{
				PiiCategories: []actions.PiiCategory{actions.EmailMasking},
			},
		},
	)
	processed, err := processor.processTraces(context.Background(), traces)
	assert.NoError(t, err)

	processedResource := processed.ResourceSpans().At(0)
	processedScope := processedResource.ScopeSpans().At(0)
	processedSpan := processedScope.Spans().At(0)
	resourceValue, _ := processedResource.Resource().Attributes().Get("resource.email")
	scopeValue, _ := processedScope.Scope().Attributes().Get("scope.email")
	spanValue, _ := processedSpan.Attributes().Get("span.email")
	spanList, _ := processedSpan.Attributes().Get("span.email_list")
	eventValue, _ := processedSpan.Events().At(0).Attributes().Get("event.email")

	assert.Equal(t, "***EMAIL***", resourceValue.Str(), "resource attributes must be masked")
	assert.Equal(t, "***EMAIL***", scopeValue.Str(), "scope attributes must be masked")
	assert.Equal(t, "***EMAIL***", spanValue.Str(), "span attributes must be masked")
	assert.Equal(t, "***EMAIL***", spanList.Slice().At(0).Str(), "slice values must remain supported")
	assert.Equal(t, "***EMAIL***", eventValue.Str(), "span-event attributes must be masked")
}
