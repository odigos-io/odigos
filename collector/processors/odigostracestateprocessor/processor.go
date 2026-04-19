package odigostracestateprocessor

import (
	"context"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/odigosattributes"
)

const odigosTraceStateKey = "odigos"

type traceStateProcessor struct {
	logger                   *zap.Logger
	categoryEnabled          bool
	traceDecidingRuleEnabled bool
}

func (p *traceStateProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpans := td.ResourceSpans().At(i)
		for j := 0; j < resourceSpans.ScopeSpans().Len(); j++ {
			scopeSpans := resourceSpans.ScopeSpans().At(j)
			for k := 0; k < scopeSpans.Spans().Len(); k++ {
				span := scopeSpans.Spans().At(k)
				p.processSpanTraceState(span)
			}
		}
	}

	return td, nil
}

func (p *traceStateProcessor) processSpanTraceState(span ptrace.Span) {
	traceState := span.TraceState().AsRaw()
	if traceState == "" {
		return
	}

	odigosValue := extractOdigosTraceStateValue(traceState)
	if odigosValue == "" {
		return
	}

	for _, entry := range strings.Split(odigosValue, ";") {
		key, value, found := strings.Cut(entry, ":")
		if !found {
			continue
		}
		switch key {
		case "c":
			if p.categoryEnabled {
				if value == "n" {
					span.Attributes().PutStr(odigosattributes.SamplingCategory, "noise")
				}
			}
		case "dr.p":
			if p.traceDecidingRuleEnabled {
				keepPercentage, err := strconv.ParseFloat(value, 64)
				if err == nil {
					span.Attributes().PutDouble(odigosattributes.SamplingTraceDecidingRuleKeepPercentage, keepPercentage)
				}
			}
		case "dr.id":
			if p.traceDecidingRuleEnabled {
				span.Attributes().PutStr(odigosattributes.SamplingTraceDecidingRuleId, value)
			}
		}
	}
}

// extractOdigosTraceStateValue extracts the value of the "odigos" key from a W3C tracestate string.
// Tracestate format: "key1=value1,key2=value2,..."
func extractOdigosTraceStateValue(traceState string) string {
	for _, entry := range strings.Split(traceState, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		k, v, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if strings.TrimSpace(k) == odigosTraceStateKey {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
