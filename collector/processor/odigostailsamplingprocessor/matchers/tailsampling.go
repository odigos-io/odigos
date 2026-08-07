package matchers

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func NewTailSamplingOperationMatcher(operation *commonapisampling.TailSamplingOperationMatcher) Matcher {
	if operation == nil {
		return anyMatcher{}
	}
	switch {
	case operation.HttpServer != nil:
		return newTailSamplingHttpServerMatcher(operation.HttpServer)
	case operation.KafkaConsumer != nil:
		return newTailSamplingKafkaConsumerMatcher(operation.KafkaConsumer)
	case operation.KafkaProducer != nil:
		return newTailSamplingKafkaProducerMatcher(operation.KafkaProducer)
	default:
		return anyMatcher{}
	}
}

type tailSamplingHttpServerMatcher struct {
	method      string
	route       string
	routePrefix string
}

func newTailSamplingHttpServerMatcher(operation *commonapisampling.TailSamplingHttpServerOperationMatcher) Matcher {
	return &tailSamplingHttpServerMatcher{
		method:      operation.Method,
		route:       operation.Route,
		routePrefix: operation.RoutePrefix,
	}
}

// Match returns true when:
// - the span is a server span.
// - it's an http span (contains http method)
// - all the attributes specified in the matcher are present on the span and match the values.
//
// Match returns false when:
// - it's not an http server span.
// - any of the attributes specified in the matcher are not present on the span.
// - any of the attributes specified in the matcher are present with a different value.
// - templated routes for spans that don't have the http.route attribute.
func (m *tailSamplingHttpServerMatcher) Match(span ptrace.Span) bool {
	if span.Kind() != ptrace.SpanKindServer {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	if !found {
		return false
	}
	if m.method != "" && !compareHttpMethod(httpMethod, m.method) {
		return false
	}

	if !matchHttpRoute(span, m.route, m.routePrefix) {
		return false
	}

	return true
}

type tailSamplingKafkaConsumerMatcher struct{}

func newTailSamplingKafkaConsumerMatcher(_ *commonapisampling.TailSamplingKafkaOperationMatcher) Matcher {
	return &tailSamplingKafkaConsumerMatcher{}
}

func (m *tailSamplingKafkaConsumerMatcher) Match(_ ptrace.Span) bool {
	return false
}

type tailSamplingKafkaProducerMatcher struct{}

func newTailSamplingKafkaProducerMatcher(_ *commonapisampling.TailSamplingKafkaOperationMatcher) Matcher {
	return &tailSamplingKafkaProducerMatcher{}
}

func (m *tailSamplingKafkaProducerMatcher) Match(_ ptrace.Span) bool {
	return false
}
