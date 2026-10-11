package urltemplate

import "regexp"

var (

	// matches any string that contains only digits or special characters
	// will catch things like "1234_567" but not anything that contains a letter
	noLettersRegex = regexp.MustCompile(`^[\d_\-!@#$%^&*()=+{}\[\]:;"'<>,.?/\\|` + "`" + `~]+$`)

	// matches UUIDs in the format 123e4567-e89b-12d3-a456-426614174000
	// these UUIDs are common in cloud systems and are often used as ids
	// they are 36 characters long and are made up of 5 groups of hexadecimal characters
	// separated by hyphens.
	// this regexp will allow any prefix OR suffix of the UUID to be matched
	// so for example: "PROCESS_123e4567-e89b-12d3-a456-426614174000" will also be matched
	uuidRegex = regexp.MustCompile(`(^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})|([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$)`)

	// Covers hex encoded values like (for example) span/trace IDs.
	// These are common as ids in cloud systems.
	//
	// To enforce the following conditions in a single Go regular expression:
	// - Only hexadecimal characters (lower or higher case) (0-9, a-f, A-F),
	// - More than 16 characters,
	// - An even number of characters
	//
	// It is considered safe as:
	// - letters are only limited to a-f (or upper case A-F), which any real word with 16 chars or more will fail.
	// - the regex will not match if the string is less than 16 chars, so things like "feed12" (all letters a-f) will not match.
	// - the regex will not match if the string is odd length (indicating it's not hex encoded) so another filter for extreme corner cases.
	//
	// Explanation (ChatGPT):
	// - (?:...) — A non-capturing group.
	// - [0-9a-fA-F]{2} — Matches exactly two hexadecimal characters.
	// - {8,} — Repeats that group 8 or more times, ensuring:
	// 	 - 8 × 2 = 16 characters minimum
	// 	 - Each repetition is of 2 characters → ensures even length.
	hexEncodedRegex = regexp.MustCompile(`^(?:[0-9a-fA-F]{2}){8,}$`)

	// assume that long numbers (7 continues digits or more) are ids.
	// even if they are found with some text (for example "INC0012686") they are treated as ids
	// it is very unlikely for a a number with so many digits to be static and meaningful.
	longNumberAnywhereRegex = regexp.MustCompile(`\d{7,}`)

	// based on example from real users
	// we want to catch dates that looks like "2025-25-04T12:00:00+0000" but possibly also other common date formats like:
	// ✅ Summary of Supported Formats (Chat GPT):
	//
	// Format	Example
	// YYYY-MM-DD	2025-12-04
	// YYYY-MM-DDTHH:MM	2025-12-04T14:55
	// YYYY-MM-DDTHH:MM:SS	2025-12-04T14:55:04
	// YYYY-MM-DDTHH:MMZ	2025-12-04T14:55Z
	// YYYY-MM-DDTHH:MM:SS+0000	2025-12-04T14:55:04+0000
	//
	// ❌ Not matched:
	// 2025/12/04 (slashes)
	// 04-12-2025 (day first)
	// 2025-12-04T14:55:04+00:00 (timezone with colon)
	// 2025-12-04T14:55:04.123Z (milliseconds)
	// 2025-12-04T14:55:04.123+0000 (millis with offset)
	datesRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}(?::\d{2})?)?(?:Z|[+-]\d{4})?$`)

	// matches email addresses
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// assume any invalid unicode character is not a static path segment
	replacementChar = regexp.MustCompile(`�`)
)

// SegmentTemplateName returns the template name for a path segment based on built-in
// heuristics (e.g. "id", "date", "email"), or "" if the segment should stay static.
func SegmentTemplateName(segment string) string {
	if datesRegex.MatchString(segment) {
		return "date"
	}

	if emailRegex.MatchString(segment) {
		return "email"
	}

	if noLettersRegex.MatchString(segment) ||
		longNumberAnywhereRegex.MatchString(segment) ||
		uuidRegex.MatchString(segment) ||
		hexEncodedRegex.MatchString(segment) ||
		replacementChar.MatchString(segment) {
		return "id"
	}

	return ""
}
