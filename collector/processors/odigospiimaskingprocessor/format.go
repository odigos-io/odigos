package odigospiimaskingprocessor

import (
	"fmt"
	"regexp"

	"github.com/odigos-io/odigos/common/api/actions"
)

// buildFormatMaskingRegexes returns the patterns that capture the value of key for the given format.
// The key is anchored on a JSON/SQL/URL boundary so substrings like "myfoo_bar" don't cross-match "foo_bar".
//
// JSON and SQL get two patterns: a quoted one that captures the whole value up to the closing quote
// (values may contain spaces and commas, e.g. a full name or a street address), and the unquoted one
// for values that are not wrapped in quotes. The patterns are applied in order, so a quoted value is
// masked in full before the unquoted pattern sees it; re-masking the resulting "****" is a no-op.
func buildFormatMaskingRegexes(key string, format actions.DataFormat) ([]*regexp.Regexp, error) {
	escapedKey := regexp.QuoteMeta(key)
	switch format {
	case actions.FormatJSON:
		// Examples (key = "user_id"):
		//   Quoted:     {"user_id": "abc 123", "name": "foo"}  -> captures "abc 123"
		//   Unquoted:   {user_id: 42, name: "foo"}             -> captures "42"
		return compileAll(
			`(?:^|[\s,{])"?`+escapedKey+`"?\s*:\s*"((?:[^"\\]|\\.)+)"?`,
			`(?:^|[\s,{])"?`+escapedKey+`"?\s*:\s*"?([^"\s,}\]]+)`,
		)
	case actions.FormatSQL:
		// Examples (key = "user_id"):
		//   Quoted:     WHERE user_id = '42 or 43' AND status = 'ok' -> captures "42 or 43"
		//   Unquoted:   WHERE user_id = 42 AND status = 'ok'         -> captures "42"
		return compileAll(
			`(?:^|[\s,(])`+escapedKey+`\s*=\s*'((?:[^'\\]|\\.)+)'?`,
			`(?:^|[\s,(])`+escapedKey+`\s*=\s*'?([^'\s,;)]+)`,
		)
	case actions.FormatResourcePath:
		// Examples (key = "orders"):
		//   Path: /api/v1/orders/abc-123 -> captures "abc-123"
		return compileAll(`(?:^|/)` + escapedKey + `/([^/\s"?&#]+)`)
	default:
		return nil, fmt.Errorf("unsupported dataFormat %q", format)
	}
}

func compileAll(patterns ...string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}
