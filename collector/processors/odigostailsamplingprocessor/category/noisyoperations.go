package category

import (
	"strings"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv_1_4_0 "go.opentelemetry.io/otel/semconv/v1.4.0"

	commonapi "github.com/odigos-io/odigos/common/api"
)

func EvaluateNoisyOperations(span ptrace.Span, noisyOperations []commonapi.WorkloadNoisyOperation) (bool, *commonapi.WorkloadNoisyOperation) {

	// aggregate the matching rules in a list.
	// there should be very few, so the length is expected to be 0 almost always,
	// 1 occassionally, and more very rarely.
	var leastPercentageRule *commonapi.WorkloadNoisyOperation

	for _, noisyOperation := range noisyOperations {

		var currentPercentage float64 = 0 // default to 0%
		if noisyOperation.PercentageAtMost != nil {
			currentPercentage = *noisyOperation.PercentageAtMost
		}

		// shortcut - we are only interested in the least percentage rule,
		// so avoid checking when unnecessary.
		// percentageAtMost as nil, means that it's the default 0%, so it's already the smallest possible.
		if leastPercentageRule != nil && (leastPercentageRule.PercentageAtMost == nil || currentPercentage >= *(leastPercentageRule.PercentageAtMost)) {
			continue
		}

		// check if the operation matches the span.
		matched := operationMatcher(noisyOperation.Operation, span)

		// at this point, we already know the current percentage is least than the one seen so far,
		// so if we have a match, we update.
		if matched {
			leastPercentageRule = &noisyOperation
		}
	}

	if leastPercentageRule != nil {
		return true, leastPercentageRule
	} else {
		return false, nil
	}
}

func operationMatcher(operation *commonapi.HeadSamplingOperationMatcher, span ptrace.Span) bool {
	if operation.HttpServer != nil {
		return operationHttpServerMatcher(operation.HttpServer, span)
	}
	if operation.HttpClient != nil {
		return operationHttpClientMatcher(operation.HttpClient, span)
	}
	return false
}

func operationHttpServerMatcher(operation *commonapi.HeadSamplingHttpServerOperationMatcher, span ptrace.Span) bool {

	if span.Kind() != ptrace.SpanKindServer {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	if !found {
		// this matcher is for http operations only, and lack of method signals no match.
		return false
	}

	// if the opeartion matcher specified a method, check it against the span method.
	if operation.Method != "" && !strings.EqualFold(operation.Method, httpMethod) {
		return false
	}

	// first check the http route attribute
	// it is assumed that the route is present at this point.
	// or that urltemplatization has enriched it.
	httpRoute, found := span.Attributes().Get(string(semconv.HTTPRouteKey))
	if found {
		return compareOperationHttpRoute(operation, httpRoute.Str())
	}

	// if the route is not present, we fallback to the path.
	// the path is not templated, and we should support it in the future.
	httpPath, found := span.Attributes().Get(string(semconv.URLPathKey))
	if found {
		return compareOperationHttpRoute(operation, httpPath.Str())
	}

	// check the old semconv http.target attribute.
	// it is not templated, and also contains the query string, which should be fixed.
	httpTarget, found := span.Attributes().Get(string(semconv_1_4_0.HTTPTargetKey))
	if found {
		return compareOperationHttpRoute(operation, httpTarget.Str())
	}

	// if no http route attribute is found, there is no match.
	return false
}

func operationHttpClientMatcher(operation *commonapi.HeadSamplingHttpClientOperationMatcher, span ptrace.Span) bool {

	// this matcher is for http client operations only, and only client spans are considered.
	if span.Kind() != ptrace.SpanKindClient {
		return false
	}

	// make sure it's an http operation by pulling the method.
	httpMethod, found := getHttpMethod(span)
	if !found {
		return false
	}

	// if the opeartion matcher specified a method, check it against the span method.
	if operation.Method != "" && !strings.EqualFold(operation.Method, httpMethod) {
		return false
	}

	// if the server address is specified, check it against the span attributes.
	if operation.ServerAddress != "" {
		serverAddress, found := span.Attributes().Get(string(semconv.ServerAddressKey))
		if found && !strings.EqualFold(operation.ServerAddress, serverAddress.Str()) {
			return false
		}
	}

	// if the url path is specified, check it against the span attributes.
	if operation.UrlPath != "" {
		urlPath, found := span.Attributes().Get(string(semconv.URLPathKey))
		if found && !strings.EqualFold(operation.UrlPath, urlPath.Str()) {
			return false
		}
	}

	return true
}

func compareOperationHttpRoute(httpServerOperation *commonapi.HeadSamplingHttpServerOperationMatcher, httpRoute string) bool {
	if httpServerOperation.Route != "" {
		return httpServerOperation.Route == httpRoute
	}
	if httpServerOperation.RoutePrefix != "" {
		return strings.HasPrefix(httpRoute, httpServerOperation.RoutePrefix)
	}
	return false
}

func getHttpMethod(span ptrace.Span) (string, bool) {
	httpMethod, found := span.Attributes().Get(string(semconv.HTTPRequestMethodKey))
	if found {
		return httpMethod.Str(), true
	}
	httpMethod, found = span.Attributes().Get(string(semconv_1_4_0.HTTPMethodKey))
	if found {
		return httpMethod.Str(), true
	}
	return "", false
}
