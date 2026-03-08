package category

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapi "github.com/odigos-io/odigos/common/api"
)

func EvaluateNoisyOperations(span ptrace.Span, noisyOperations []commonapi.WorkloadNoisyOperation) (bool, *commonapi.WorkloadNoisyOperation) {

	// aggregate the matching rules in a list.
	// there should be very few, so the length is expected to be 0 almost always,
	// 1 occassionally, and more very rarely.
	var leastPercentageRule *commonapi.WorkloadNoisyOperation

	for _, noisyOperation := range noisyOperations {

		currentPercentage := GetPercentageOrDefault(noisyOperation.PercentageAtMost, 0.0)

		// shortcut - we are only interested in the least percentage rule,
		// so avoid checking when unnecessary.
		// percentageAtMost as nil, means that it's the default 0%, so it's already the smallest possible.
		if leastPercentageRule != nil && (leastPercentageRule.PercentageAtMost == nil || currentPercentage >= *(leastPercentageRule.PercentageAtMost)) {
			continue
		}

		// check if the operation matches the span.
		matched := matchers.HeadSamplingOperationMatcher(noisyOperation.Operation, span)

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
