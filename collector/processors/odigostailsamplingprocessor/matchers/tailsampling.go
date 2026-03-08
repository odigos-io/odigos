package matchers

import (
	commonapisanpling "github.com/odigos-io/odigos/common/api/sampling"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TailSamplingOperationMatcher(operation *commonapisanpling.TailSamplingOperationMatcher, span ptrace.Span) bool {
	if operation == nil {
		// if operation is not specified, it will match any operation.
		return true
	}
	if operation.HttpServer != nil {
		return operationHttpServerMatcher(operation.HttpServer, span)
	}
	if operation.KafkaConsumer != nil {
		return operationKafkaConsumerMatcher(operation.KafkaConsumer, span)
	}
	if operation.KafkaProducer != nil {
		return operationKafkaProducerMatcher(operation.KafkaProducer, span)
	}
	// no operation type specified, match any.
	return true
}

// given a span and a http server operation matcher, will attempt to match the span to the matcher.
//
// will return true when:
// - the span is a server span.
// - it's an http span (contains http method)
// - all the attributes specified in the matcher are present on the span and match the values.
//
// will return false when:
// - it's not an http server span.
// - any of the attributes specified in the matcher are not present on the span.
// - any of the attributes specified in the matcher are presence with a different value.
// - templated routes for spans that don't have the http.route attribute.
func operationHttpServerMatcher(operation *commonapisanpling.HttpServerOperationMatcher, span ptrace.Span) bool {
	if span.Kind() != ptrace.SpanKindServer {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	if !found {
		// this matcher is for http operations only, and lack of method signals no match.
		return false
	}
	if operation.Method != "" && !compareHttpMethod(httpMethod, operation.Method) {
		return false
	}

	if !matchHttpRoute(span, operation.Route, operation.RoutePrefix) {
		return false
	}

	return true
}

func operationKafkaConsumerMatcher(operation *commonapisanpling.KafkaOperationMatcher, span ptrace.Span) bool {
	return false
}

func operationKafkaProducerMatcher(operation *commonapisanpling.KafkaOperationMatcher, span ptrace.Span) bool {
	return false
}
