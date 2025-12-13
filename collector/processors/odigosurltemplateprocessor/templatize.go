package odigosurltemplateprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

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
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// assume any invalid unicode character is not a static path segment
	replacementChar = regexp.MustCompile(`�`)
)

type RulePathSegment struct {

	// if wildcard is true, it mean that tthis path segment always matches the path segment.
	// the content of the path will not be templated,
	// and it's the user responsibility to ensure this value has low cardinality.
	Wildcard bool

	// if this rule path segment is a static string (e.g. "users"), this value will be non-empty
	StaticString string

	// it this rule segment path is replaced with templated name, the TemplateName will be non-empty
	TemplateName string

	// for templated segment names, a user can also include an optional regexp
	// which must match for the rule to be applied.
	// it templatedName is unset, the regexp will be used to match the path segment but not template them.
	RegexpPattern *regexp.Regexp
}

type TemplatizationRule []RulePathSegment

func parseRuleTemplateString(ruleTemplateString string) (string, *regexp.Regexp, error) {
	// template string is in the format name:optional-regexp.
	// for example: {userId:[0-9]+} or {userId}
	// if the regexp is not provided, it will be nil

	parts := strings.SplitN(ruleTemplateString, ":", 2)
	if len(parts) == 0 {
		return "", nil, errors.New("invalid rule template string")
	}
	templateName := strings.TrimSpace(parts[0])
	if templateName == "" {
		templateName = "id" // default to name id if not provided
	}
	var regexpPattern *regexp.Regexp
	if len(parts) == 2 {
		regexpString := strings.TrimSpace(parts[1])
		if regexpString == "" {
			return "", nil, errors.New("invalid rule template string. regexp is empty")
		}
		var err error
		regexpPattern, err = regexp.Compile(regexpString)
		if err != nil {
			return "", nil, fmt.Errorf("invalid rule template string. regexp is invalid: %w", err)
		}
	}
	return templateName, regexpPattern, nil
}

// parseRegexPattern checks if a segment starts with "regex:" prefix
// and returns the regex pattern if found, or nil if not
func parseRegexPattern(segment string) (*regexp.Regexp, string, error) {
	// Check if segment starts with "regex:" prefix
	if strings.HasPrefix(segment, "regex:") && len(segment) > 6 {
		regexString := segment[6:] // len("regex:") = 6
		regexpPattern, err := regexp.Compile(regexString)
		if err != nil {
			return nil, "", fmt.Errorf("invalid regexp pattern %q: %w", regexString, err)
		}
		return regexpPattern, "", nil
	}
	return nil, segment, nil
}

func parseUserInputRuleString(userInputRule string) ([]RulePathSegment, error) {
	segments := strings.Split(userInputRule, "/")
	if strings.HasPrefix(userInputRule, "/") {
		// if the rule starts with a /, remove it
		// this is to avoid empty string in the first segment
		segments = segments[1:]
	}

	ruleSegments := make([]RulePathSegment, len(segments))

	for i, segment := range segments {
		// if the segment looks like {text}, then it's a template
		if segment == "*" {
			ruleSegments[i] = RulePathSegment{
				Wildcard: true,
			}
			continue
		} else if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			// remove the curly braces
			templatizationRule := segment[1 : len(segment)-1]
			// parse the template name and optional regexp
			templateName, regexpPattern, err := parseRuleTemplateString(templatizationRule)
			if err != nil {
				return nil, err
			}
			ruleSegments[i] = RulePathSegment{
				TemplateName:  templateName,
				RegexpPattern: regexpPattern,
			}
		} else {
			// Check if it's a regex pattern prefixed with "regex:"
			regexpPattern, staticString, err := parseRegexPattern(segment)
			if err != nil {
				return nil, err
			}
			if regexpPattern != nil {
				// it's a regex pattern for untemplatized section matching
				ruleSegments[i] = RulePathSegment{
					RegexpPattern: regexpPattern,
				}
			} else {
				// otherwise, it's a static string
				ruleSegments[i] = RulePathSegment{
					StaticString: staticString,
				}
			}
		}
	}

	return ruleSegments, nil
}

func attemptTemplateWithRule(pathSegments []string, ruleSegments TemplatizationRule) (string, bool) {
	// already verified that the len of the lists match pre calling this function

	for i, pathSegment := range pathSegments {
		ruleSegment := ruleSegments[i]

		if ruleSegment.Wildcard {
			// if this segment is a wildcard, it always matches the path segment
			continue
		}

		// if this segment is a static string, it must match the path segment exactly
		if ruleSegment.StaticString != "" && ruleSegment.StaticString != pathSegment {
			// if the static string does not match, we can't use this rule
			return "", false
		}

		if ruleSegment.TemplateName != "" && ruleSegment.RegexpPattern != nil {
			// if this segment is a templated segment, it must match the regexp pattern
			if !ruleSegment.RegexpPattern.MatchString(pathSegment) {
				// if the regexp pattern does not match, we can't use this rule
				return "", false
			}
		} else if ruleSegment.RegexpPattern != nil {
			// if a regexp pattern is provided, use it for matching
			if !ruleSegment.RegexpPattern.MatchString(pathSegment) {
				// if the regexp pattern does not match, we can't use this rule
				return "", false
			}
		}
	}

	result := make([]string, 0, len(ruleSegments))
	for i, segment := range ruleSegments {
		if segment.TemplateName != "" {
			result = append(result, "{"+segment.TemplateName+"}")
		} else if segment.Wildcard || segment.RegexpPattern != nil {
			// untemplated value, keep whatever is in the path (assumes low cardinality)
			result = append(result, pathSegments[i])
		} else {
			result = append(result, segment.StaticString)
		}
	}

	return strings.Join(result, "/"), true
}

// return the name to use for templatization "id" / "date" etc which will be embedded in the template
// as {id} / {date} etc
// empty string as return value means that the segment is not a templated id
func getSegmentTemplatizationString(segment string, customIds []internalCustomIdConfig) string {

	// check if the segment matches any of the custom ids regexp
	for _, customRegexp := range customIds {
		if customRegexp.Regexp.MatchString(segment) {
			return customRegexp.Name
		}
	}

	if datesRegex.MatchString(segment) {
		return "date"
	}

	if emailRegex.MatchString(segment) {
		return "email"
	}

	// check if the segment is a number or uuid
	if noLettersRegex.MatchString(segment) ||
		longNumberAnywhereRegex.MatchString(segment) ||
		uuidRegex.MatchString(segment) ||
		hexEncodedRegex.MatchString(segment) ||
		replacementChar.MatchString(segment) {
		return "id"
	}

	return ""
}

// This function will replace all segments that matches a number or uuid with "{id}"
func defaultTemplatizeURLPath(pathSegments []string, customIdsRegexp []internalCustomIdConfig) (string, bool) {
	templated := false
	// avoid modifying the original segments slice
	templatizedSegments := make([]string, len(pathSegments))
	for i, segment := range pathSegments {
		if templateName := getSegmentTemplatizationString(segment, customIdsRegexp); templateName != "" {
			templatizedSegments[i] = "{" + templateName + "}"
			templated = true
		} else {
			templatizedSegments[i] = segment
		}
	}
	if !templated {
		return "", false
	} else {
		templatedPath := strings.Join(templatizedSegments, "/")
		return templatedPath, true
	}
}
