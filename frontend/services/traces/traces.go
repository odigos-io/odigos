package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

// JaegerGetTracesOptions represents the options for fetching traces
type JaegerGetTracesOptions struct {
	ServiceName   string  `json:"serviceName"`
	SearchDepth   int     `json:"searchDepth"`
	StartTimeMin  string  `json:"startTimeMin"` // RFC-3339ns format (YYYY-MM-DDTHH:MM:SSZ)
	StartTimeMax  string  `json:"startTimeMax"` // RFC-3339ns format (YYYY-MM-DDTHH:MM:SSZ)
	OperationName *string `json:"operationName,omitempty"`
	DurationMin   *int    `json:"durationMin,omitempty"`
	DurationMax   *int    `json:"durationMax,omitempty"`
}

// JaegerAttributeValue represents the different types of attribute values
type JaegerAttributeValue struct {
	StringValue *string `json:"stringValue,omitempty"`
	IntValue    *string `json:"intValue,omitempty"` // Can be string or number
	DoubleValue *int    `json:"doubleValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
}

// JaegerAttribute represents a key-value attribute
type JaegerAttribute struct {
	Key   string               `json:"key"`
	Value JaegerAttributeValue `json:"value"`
}

// JaegerResource represents resource information with attributes
type JaegerResource struct {
	Attributes             []JaegerAttribute `json:"attributes"`
	DroppedAttributesCount *int              `json:"droppedAttributesCount,omitempty"`
}

// JaegerInstrumentationScope represents OpenTelemetry instrumentation scope
type JaegerInstrumentationScope struct {
	Name                   string            `json:"name"`
	Version                *string           `json:"version,omitempty"`
	Attributes             []JaegerAttribute `json:"attributes,omitempty"`
	DroppedAttributesCount *int              `json:"droppedAttributesCount,omitempty"`
}

// JaegerSpanStatus represents the status of a span
type JaegerSpanStatus struct {
	Code    *int    `json:"code,omitempty"`
	Message *string `json:"message,omitempty"`
}

// JaegerSpanEvent represents an event within a span
type JaegerSpanEvent struct {
	TimeUnixNano           string            `json:"timeUnixNano"`
	Name                   string            `json:"name"`
	Attributes             []JaegerAttribute `json:"attributes,omitempty"`
	DroppedAttributesCount *int              `json:"droppedAttributesCount,omitempty"`
}

// JaegerSpanLink represents a link to another span
type JaegerSpanLink struct {
	TraceID                string            `json:"traceId"`
	SpanID                 string            `json:"spanId"`
	TraceState             *string           `json:"traceState,omitempty"`
	Attributes             []JaegerAttribute `json:"attributes,omitempty"`
	DroppedAttributesCount *int              `json:"droppedAttributesCount,omitempty"`
}

// JaegerSpan represents an individual trace span
type JaegerSpan struct {
	TraceID                string            `json:"traceId"`
	SpanID                 string            `json:"spanId"`
	TraceState             *string           `json:"traceState,omitempty"`
	ParentSpanID           *string           `json:"parentSpanId,omitempty"`
	Name                   string            `json:"name"`
	Kind                   int               `json:"kind"` // SpanKind enum: 0=UNSPECIFIED, 1=INTERNAL, 2=SERVER, 3=CLIENT, 4=PRODUCER, 5=CONSUMER
	StartTimeUnixNano      string            `json:"startTimeUnixNano"`
	EndTimeUnixNano        string            `json:"endTimeUnixNano"`
	Attributes             []JaegerAttribute `json:"attributes,omitempty"`
	DroppedAttributesCount *int              `json:"droppedAttributesCount,omitempty"`
	Events                 []JaegerSpanEvent `json:"events,omitempty"`
	DroppedEventsCount     *int              `json:"droppedEventsCount,omitempty"`
	Links                  []JaegerSpanLink  `json:"links,omitempty"`
	DroppedLinksCount      *int              `json:"droppedLinksCount,omitempty"`
	Status                 *JaegerSpanStatus `json:"status,omitempty"`
}

// JaegerScopeSpan groups spans by instrumentation scope
type JaegerScopeSpan struct {
	Scope     JaegerInstrumentationScope `json:"scope"`
	Spans     []JaegerSpan               `json:"spans"`
	SchemaURL *string                    `json:"schemaUrl,omitempty"`
}

// JaegerResourceSpan groups scope spans by resource
type JaegerResourceSpan struct {
	Resource   JaegerResource    `json:"resource"`
	ScopeSpans []JaegerScopeSpan `json:"scopeSpans"`
	SchemaURL  *string           `json:"schemaUrl,omitempty"`
}

// JaegerGetTracesResponse contains the resource spans
type JaegerGetTracesResponse struct {
	ResourceSpans []JaegerResourceSpan `json:"resourceSpans"`
}

// JaegerGetTracesResult represents the response from the traces API
type JaegerGetTracesResult struct {
	Result JaegerGetTracesResponse `json:"result"`
}

// buildQueryString creates URL query parameters from options
func buildQueryString(options JaegerGetTracesOptions) string {
	params := url.Values{
		"query.service_name":   {options.ServiceName},
		"query.start_time_min": {options.StartTimeMin},
		"query.start_time_max": {options.StartTimeMax},
		"query.search_depth":   {strconv.Itoa(options.SearchDepth)},
	}

	// Only add duration parameters if they are not nil
	if options.OperationName != nil {
		params.Set("query.operation_name", *options.OperationName)
	}
	if options.DurationMin != nil {
		params.Set("query.duration_min", strconv.Itoa(*options.DurationMin))
	}
	if options.DurationMax != nil {
		params.Set("query.duration_max", strconv.Itoa(*options.DurationMax))
	}

	return params.Encode()
}

// GetTraces fetches traces from the Jaeger API
func GetTraces(ctx context.Context, jaegerURL string, options JaegerGetTracesOptions) ([]JaegerResourceSpan, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	queryString := buildQueryString(options)
	requestURL := fmt.Sprintf("%s/api/v3/traces?%s", jaegerURL, queryString)

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	var tracesResponse JaegerGetTracesResult
	if err := json.NewDecoder(resp.Body).Decode(&tracesResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tracesResponse.Result.ResourceSpans, nil
}

// convertTraces converts OpenTelemetry format to Jaeger format
func ConvertTraces(resourceSpans []JaegerResourceSpan) []*model.Trace {
	// Group spans by traceID
	traceMap := make(map[string]*model.Trace)

	for _, resourceSpan := range resourceSpans {
		// Create process from resource
		process := convertResourceToProcess(resourceSpan.Resource)

		// Use serviceName as processID instead of generic counter
		processID := process.ServiceName

		for _, scopeSpan := range resourceSpan.ScopeSpans {
			// Add scope information to process tags if available
			if scopeSpan.Scope.Name != "" {
				scopeTags := []*model.TraceTag{
					{Key: "otel.scope.name", Type: "string", Value: scopeSpan.Scope.Name},
				}
				if scopeSpan.Scope.Version != nil {
					scopeTags = append(scopeTags, &model.TraceTag{Key: "otel.scope.version", Type: "string", Value: *scopeSpan.Scope.Version})
				}
				process.Tags = append(process.Tags, scopeTags...)
			}

			for _, jaegerSpan := range scopeSpan.Spans {
				traceID := jaegerSpan.TraceID

				// Initialize trace if not exists
				if _, exists := traceMap[traceID]; !exists {
					traceMap[traceID] = &model.Trace{
						TraceID:   traceID,
						Spans:     []*model.TraceSpan{},
						Processes: []*model.TraceProcess{},
						Warnings:  "",
					}
				}

				// Add process to trace if not already added
				processExists := false
				for _, existingProcess := range traceMap[traceID].Processes {
					if existingProcess.ServiceName == process.ServiceName {
						processExists = true
						break
					}
				}
				if !processExists {
					traceMap[traceID].Processes = append(traceMap[traceID].Processes, process)
				}

				// Convert span
				span := convertSpanFromJaeger(jaegerSpan, processID, scopeSpan.Scope)
				traceMap[traceID].Spans = append(traceMap[traceID].Spans, span)
			}
		}
	}

	// Convert map to slice
	traces := make([]*model.Trace, 0, len(traceMap))
	for _, trace := range traceMap {
		traces = append(traces, trace)
	}

	return traces
}

// convertResourceToProcess converts an OpenTelemetry Resource to a Jaeger Process
func convertResourceToProcess(resource JaegerResource) *model.TraceProcess {
	process := &model.TraceProcess{
		ServiceName: "unknown", // default
		Tags:        []*model.TraceTag{},
	}

	// Convert resource attributes to process tags
	for _, attr := range resource.Attributes {
		if attr.Key == "service.name" && attr.Value.StringValue != nil {
			process.ServiceName = *attr.Value.StringValue
		} else {
			tag := convertAttributeToTag(attr)
			process.Tags = append(process.Tags, tag)
		}
	}

	return process
}

// convertSpanFromJaeger converts an OpenTelemetry Span to a Jaeger Span
func convertSpanFromJaeger(span JaegerSpan, processID string, scope JaegerInstrumentationScope) *model.TraceSpan {
	jaegerSpan := &model.TraceSpan{
		TraceID:       span.TraceID,
		SpanID:        span.SpanID,
		OperationName: span.Name,
		ProcessID:     processID,
		StartTime:     int(convertNanosToMicros(span.StartTimeUnixNano)),
		Duration:      int(convertNanosToMicros(span.EndTimeUnixNano) - convertNanosToMicros(span.StartTimeUnixNano)),
		Tags:          []*model.TraceTag{},
		Logs:          []*model.TraceLog{},
		References:    []*model.TraceReference{},
		Warnings:      "",
	}

	// Convert parent span to reference
	if span.ParentSpanID != nil {
		jaegerSpan.References = append(jaegerSpan.References, &model.TraceReference{
			RefType: "CHILD_OF",
			TraceID: span.TraceID,
			SpanID:  *span.ParentSpanID,
		})
	}

	// Convert attributes to tags
	for _, attr := range span.Attributes {
		tag := convertAttributeToTag(attr)
		jaegerSpan.Tags = append(jaegerSpan.Tags, tag)
	}

	// Add span kind as tag
	spanKindMap := map[int]string{
		0: "unspecified",
		1: "internal",
		2: "server",
		3: "client",
		4: "producer",
		5: "consumer",
	}
	if kindStr, exists := spanKindMap[span.Kind]; exists {
		jaegerSpan.Tags = append(jaegerSpan.Tags, &model.TraceTag{
			Key:   "span.kind",
			Type:  "string",
			Value: kindStr,
		})
	}

	// Convert events to logs
	for _, event := range span.Events {
		fields := []*model.TraceTag{
			{Key: "event", Type: "string", Value: event.Name},
		}
		for _, attr := range event.Attributes {
			tag := convertAttributeToTag(attr)
			fields = append(fields, tag)
		}
		jaegerSpan.Logs = append(jaegerSpan.Logs, &model.TraceLog{
			Timestamp: int(convertNanosToMicros(event.TimeUnixNano)),
			Fields:    fields,
		})
	}

	// Add status information as tags
	if span.Status != nil {
		if span.Status.Code != nil {
			jaegerSpan.Tags = append(jaegerSpan.Tags, &model.TraceTag{
				Key:   "otel.status_code",
				Type:  "int64",
				Value: fmt.Sprintf("%d", *span.Status.Code),
			})
		}
		if span.Status.Message != nil {
			jaegerSpan.Tags = append(jaegerSpan.Tags, &model.TraceTag{
				Key:   "otel.status_description",
				Type:  "string",
				Value: *span.Status.Message,
			})
		}
	}

	return jaegerSpan
}

// convertAttributeToTag converts an OpenTelemetry Attribute to a Jaeger Tag
func convertAttributeToTag(attr JaegerAttribute) *model.TraceTag {
	tag := &model.TraceTag{Key: attr.Key}

	switch {
	case attr.Value.StringValue != nil:
		tag.Type = "string"
		tag.Value = *attr.Value.StringValue
	case attr.Value.IntValue != nil:
		tag.Type = "int"
		tag.Value = *attr.Value.IntValue
	case attr.Value.DoubleValue != nil:
		tag.Type = "float64"
		tag.Value = fmt.Sprintf("%d", *attr.Value.DoubleValue)
	case attr.Value.BoolValue != nil:
		tag.Type = "bool"
		tag.Value = fmt.Sprintf("%t", *attr.Value.BoolValue)
	default:
		tag.Type = "string"
		tag.Value = fmt.Sprintf("%v", *attr.Value.BoolValue)
	}

	return tag
}

// convertNanosToMicros converts nanosecond timestamp to microseconds
func convertNanosToMicros(nanosStr string) int64 {
	nanos, err := strconv.ParseInt(nanosStr, 10, 64)
	if err != nil {
		return 0
	}
	return nanos / 1000 // Convert nanoseconds to microseconds
}
