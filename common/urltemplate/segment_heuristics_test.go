package urltemplate

import (
	"sort"
	"strings"
	"testing"
)

// The five heuristics behind the "id" template name are OR'd together in a single
// condition, so a fixture that matches two of them cannot see either one being dropped.
// Naming them lets every case below be pinned to the one heuristic it exercises.
var sgIDHeuristics = map[string]func(string) bool{
	"noLetters":       noLettersRegex.MatchString,
	"longNumber":      longNumberAnywhereRegex.MatchString,
	"uuid":            uuidRegex.MatchString,
	"hexEncoded":      hexEncodedRegex.MatchString,
	"replacementChar": replacementChar.MatchString,
}

// The template names README.md advertises as the complete output range of
// SegmentTemplateName.
var sgDocumentedNames = []string{"", "date", "email", "id"}

type sgCase struct {
	segment string
	want    string
	// isolates names the single "id" heuristic this segment is expected to match, and is
	// empty for segments that are not meant to isolate one.
	isolates string
	// alsoMatchesID marks a segment that a date/email branch claims, and that an "id"
	// heuristic would also match - so the case pins the order of the branches.
	alsoMatchesID bool
	why           string
}

var sgCases = []sgCase{
	// One case per "id" heuristic, each matching only that heuristic. Several of these
	// segments are the examples the heuristics' own comments give.
	{segment: "1234_567", want: "id", isolates: "noLetters", why: "digits and separators only, no 7-digit run"},
	{segment: "1.2.3", want: "id", isolates: "noLetters", why: "a bare semver segment is digits and dots only"},
	{segment: "INC0012686", want: "id", isolates: "longNumber", why: "a long digit run inside text"},
	{segment: "INC001268", want: "", why: "six digits is under the seven digit floor the comment sets"},
	{segment: "f47ac10b-58cc-4372-a567-0e02b2c3d479", want: "id", isolates: "uuid", why: "canonical UUID, longest digit run is 4"},
	{segment: "F47AC10B-58CC-4372-A567-0E02B2C3D479", want: "id", isolates: "uuid", why: "UUIDs are matched case-insensitively"},
	{segment: "PROCESS_f47ac10b-58cc-4372-a567-0e02b2c3d479", want: "id", isolates: "uuid", why: "the comment promises a prefixed UUID still matches"},
	{segment: "f47ac10b-58cc-4372-a567-0e02b2c3d479_SUFFIX", want: "id", isolates: "uuid", why: "the comment promises a suffixed UUID still matches"},
	{segment: "f47ac10b-58cc-4372-a567-0e02b2c3d47", want: "",
		why: "the comment specifies five groups of fixed length; an 11 character final group is not a UUID"},
	{segment: "deadbeefdeadbeef", want: "id", isolates: "hexEncoded", why: "16 hex characters, all letters"},
	{segment: "ABCDEFABCDEFABCD", want: "id", isolates: "hexEncoded", why: "hex is matched case-insensitively"},
	{segment: "caf\uFFFD", want: "id", isolates: "replacementChar", why: "a lossy decode left a replacement character"},
	{segment: "caf" + string([]byte{0xc3}), want: "id", isolates: "replacementChar",
		why: "regexp decodes an invalid byte as U+FFFD, so a raw truncated sequence matches too"},

	// The date and email branches run before the "id" heuristics, and both of these
	// segments would otherwise be classified as "id".
	{segment: "2025-12-04", want: "date", alsoMatchesID: true, why: "digits and hyphens only, so noLetters claims it too"},
	{segment: "user1234567@example.com", want: "email", alsoMatchesID: true, why: "the local part carries a long digit run"},

	// The date formats the heuristic's comment lists as supported.
	{segment: "2025-12-04T14:55", want: "date"},
	{segment: "2025-12-04T14:55:04", want: "date"},
	{segment: "2025-12-04T14:55Z", want: "date"},
	{segment: "2025-12-04T14:55:04+0000", want: "date"},
	{segment: "2025-25-04T12:00:00+0000", want: "date", why: "month and day are not range checked"},

	// The date formats that comment lists as unsupported. A colon in the offset, or
	// milliseconds, leaves the segment static rather than falling through to "id".
	{segment: "2025-12-04T14:55:04+00:00", want: ""},
	{segment: "2025-12-04T14:55:04.123Z", want: ""},
	{segment: "2025-12-04T14:55:04.123+0000", want: ""},
	// Also unsupported as dates, but these two carry no letters, so they come back as
	// "id" instead of staying static.
	{segment: "2025/12/04", want: "id", isolates: "noLetters"},
	{segment: "04-12-2025", want: "id", isolates: "noLetters"},
	{segment: "order-2025-12-04", want: "", why: "a date has to span the whole segment to be one"},

	{segment: "user@example.com", want: "email"},
	{segment: "u@e.io", want: "email", why: "a two character TLD is the shortest one accepted"},
	{segment: "a@b.c", want: "", why: "a single character TLD is below that floor"},
	{segment: "user@localhost", want: "", why: "an address with no dotted TLD is not matched"},
	{segment: "mailto:user@example.com", want: "", why: "an address has to span the whole segment"},

	// The boundaries the hex comment reasons about: below 16 characters, and an odd
	// length, are both rejected so that real words are not read as hex.
	{segment: "abcdefabcdefab", want: "", why: "14 hex characters is under the 16 character floor"},
	{segment: "abcdefabcdefabcd", want: "id", isolates: "hexEncoded", why: "16 is the floor"},
	{segment: "abcdefabcdefabcde", want: "", why: "17 characters is an odd length"},
	{segment: "feed12", want: "", why: "the comment's example of a short all-hex word"},

	// Real path segments have to stay static, including non-ASCII words, which must not
	// be read as replacement characters.
	{segment: "users", want: ""},
	{segment: "api", want: ""},
	{segment: "health", want: ""},
	{segment: "v1", want: ""},
	{segment: "v2beta1", want: ""},
	{segment: "v1.2.3", want: "", why: "a leading letter keeps the semver segment static"},
	{segment: "\u00FCrl", want: "", why: "valid multi-byte UTF-8 is not a replacement character"},
	{segment: "\u5065\u5EB7", want: ""},
	{segment: "", want: ""},
}

// sgMatchingIDHeuristics reports, sorted, which of the "id" heuristics match segment.
func sgMatchingIDHeuristics(segment string) []string {
	var matched []string
	for name, match := range sgIDHeuristics {
		if match(segment) {
			matched = append(matched, name)
		}
	}
	sort.Strings(matched)
	return matched
}

func TestSegmentTemplateNameClassifiesSegments(t *testing.T) {
	produced := map[string]bool{}
	for _, tt := range sgCases {
		got := SegmentTemplateName(tt.segment)
		if got != tt.want {
			reason := ""
			if tt.why != "" {
				reason = " - " + tt.why
			}
			t.Errorf("SegmentTemplateName(%q) = %q, want %q%s", tt.segment, got, tt.want, reason)
		}
		produced[got] = true
	}

	// README.md advertises exactly these names, so a heuristic that starts returning
	// something else, or a name that stops being reachable, is a documentation break.
	names := make([]string, 0, len(produced))
	for name := range produced {
		names = append(names, name)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != strings.Join(sgDocumentedNames, ",") {
		t.Errorf("template names produced = %q, want the documented %q", names, sgDocumentedNames)
	}
}

// Every "id" heuristic has to be the only reason some segment is classified, otherwise
// dropping it from the OR chain changes nothing and the suite cannot see it go.
func TestEveryIDHeuristicIsTheSoleReasonForSomeSegment(t *testing.T) {
	isolated := map[string]bool{}
	for _, tt := range sgCases {
		if tt.isolates == "" {
			continue
		}
		if _, ok := sgIDHeuristics[tt.isolates]; !ok {
			t.Fatalf("case %q names unknown heuristic %q", tt.segment, tt.isolates)
		}
		matched := sgMatchingIDHeuristics(tt.segment)
		if len(matched) != 1 || matched[0] != tt.isolates {
			t.Errorf("%q is matched by %v, want only %q - it no longer isolates that heuristic",
				tt.segment, matched, tt.isolates)
			continue
		}
		isolated[tt.isolates] = true
	}

	for name := range sgIDHeuristics {
		if !isolated[name] {
			t.Errorf("no case isolates the %q heuristic, so dropping it would go unnoticed", name)
		}
	}
}

// The date and email checks come first, and only a segment an "id" heuristic would also
// claim can prove it.
func TestDateAndEmailAreCheckedBeforeTheIDHeuristics(t *testing.T) {
	contested := 0
	for _, tt := range sgCases {
		if !tt.alsoMatchesID {
			continue
		}
		contested++
		if matched := sgMatchingIDHeuristics(tt.segment); len(matched) == 0 {
			t.Errorf("%q matches no id heuristic, so it cannot pin the branch order", tt.segment)
		}
		if got := SegmentTemplateName(tt.segment); got != tt.want {
			t.Errorf("SegmentTemplateName(%q) = %q, want %q", tt.segment, got, tt.want)
		}
	}
	if contested < 2 {
		t.Errorf("found %d contested segments, want one for the date branch and one for email", contested)
	}
}
