package odigospiimaskingprocessor

import (
	"regexp"

	"github.com/odigos-io/odigos/common/api/actions"
)

type categoryMask struct {
	maskedValue string
	patterns    []*regexp.Regexp
}

var categoryMasks = map[actions.PiiCategory]categoryMask{
	actions.CreditCardMasking: {
		maskedValue: "***CREDIT_CARD***",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`4[0-9]{12}(?:[0-9]{3})?`), // Visa
			regexp.MustCompile(`5[1-5][0-9]{14}`),         // MasterCard
		},
	},
	actions.EmailMasking: {
		maskedValue: "***EMAIL***",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		},
	},
	actions.JwtMasking: {
		maskedValue: "***JWT***",
		patterns: []*regexp.Regexp{
			// JWTs encode a JSON header, so the first two segments typically start with "eyJ".
			regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`),
		},
	},
	actions.UuidMasking: {
		maskedValue: "***UUID***",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),
		},
	},
}

func maskCategory(category actions.PiiCategory, value string) (string, bool) {
	mask, ok := categoryMasks[category]
	if !ok {
		return value, false
	}

	result := value
	changed := false
	for _, pattern := range mask.patterns {
		replaced := pattern.ReplaceAllString(result, mask.maskedValue)
		if replaced != result {
			result = replaced
			changed = true
		}
	}
	return result, changed
}
