package odigospiimaskingprocessor

import (
	"fmt"
	"regexp"

	"github.com/odigos-io/odigos/common/api/actions"
)

// buildFormatMaskingRegex returns a pattern that captures the value of key for the given format.
// The key is anchored on a JSON/SQL/URL boundary so substrings like "myfoo_bar" don't cross-match "foo_bar".
func buildFormatMaskingRegex(key string, format actions.DataFormat) (*regexp.Regexp, error) {
	escapedKey := regexp.QuoteMeta(key)
	switch format {
	case actions.FormatJSON:
		// Examples (key = "user_id"):
		//   Quoted:     {"user_id": "abc123", "name": "foo"}   -> captures "abc123"
		//   Unquoted:   {user_id: 42, name: "foo"}             -> captures "42"
		return regexp.Compile(`(?:^|[\s,{])"?` + escapedKey + `"?\s*:\s*(?:"((?:\\.|[^"\\])*)"|([^"\s,}\]]+))`)
	case actions.FormatSQL:
		// Examples (key = "user_id"):
		//   Quoted:     WHERE user_id = '42' AND status = 'ok'  -> captures "42"
		return regexp.Compile(`(?:^|[\s,(])` + escapedKey + `\s*=\s*(?:'((?:''|[^'])*)'|([^'\s,;)]+))`)
	case actions.FormatResourcePath:
		// Examples (key = "orders"):
		//   Path: /api/v1/orders/abc-123 -> captures "abc-123"
		return regexp.Compile(`(?:^|/)` + escapedKey + `/([^/\s"?&#]+)`)
	default:
		return nil, fmt.Errorf("unsupported dataFormat %q", format)
	}
}
