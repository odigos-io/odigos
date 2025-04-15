package odigosurltemplateprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	onlyDigitsRegex = regexp.MustCompile(`^\d+$`)

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
	// - Only lowercase hexadecimal characters (0-9 and a-f),
	// - More than 16 characters,
	// - An even number of characters
	//
	// It is considered safe as:
	// - letters are only limited to lowercase a-f, which any real word with 16 chars or more will fail.
	// - the regex will not match if the string is less than 16 chars, so things like "feed12" (all letters a-f) will not match.
	// - the regex will not match if the string is odd length (indicating it's not hex encoded) so another filter for extreme corner cases.
	//
	// Explanation (ChatGPT):
	// - (?:...) — A non-capturing group.
	// - [0-9a-f]{2} — Matches exactly two hexadecimal characters.
	// - {8,} — Repeats that group 8 or more times, ensuring:
	// 	 - 8 × 2 = 16 characters minimum
	// 	 - Each repetition is of 2 characters → ensures even length.
	hexEncodedRegex = regexp.MustCompile(`^(?:[0-9a-f]{2}){8,}$`)

	// assume that long numbers (more than 8 digits) are ids.
	// even if they are found with some text (for example "INC001268637") they are treated as ids
	// it is very unlikely for a a number with so many digits to be static and meaningful.
	longNumberAnywhereRegex = regexp.MustCompile(`\d{9,}`)
)

type RulePathSegment struct {

	// if this rule path segment is a static string (e.g. "users"), this value will be non-empty
	StaticString string

	// it this rule segment path is replaced with templated name, the TemplateName will be non-empty
	TemplateName string
	// for templated segment names, a user can also include an optional regexp
	// which must match for the rule to be applied.
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
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
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
			// otherwise, it's a static string
			ruleSegments[i] = RulePathSegment{
				StaticString: segment,
			}
		}
	}

	return ruleSegments, nil
}

func attemptTemplateWithRule(pathSegments []string, ruleSegments TemplatizationRule) (string, bool) {

	// all segments must match, so they have to be the same length
	if len(pathSegments) != len(ruleSegments) {
		return "", false
	}

	for i, pathSegment := range pathSegments {
		ruleSegment := ruleSegments[i]

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
		}
	}

	result := make([]string, 0, len(ruleSegments))
	for _, segment := range ruleSegments {
		if segment.TemplateName != "" {
			result = append(result, "{"+segment.TemplateName+"}")
		} else {
			result = append(result, segment.StaticString)
		}
	}

	return strings.Join(result, "/"), true
}

// This function will replace all segments that matches a number or uuid with "{id}"
func defaultTemplatizeURLPath(pathSegments []string) (string, bool) {
	templated := false
	// avoid modifying the original segments slice
	templatizedSegments := make([]string, len(pathSegments))
	for i, segment := range pathSegments {
		if onlyDigitsRegex.MatchString(segment) ||
			longNumberAnywhereRegex.MatchString(segment) ||
			uuidRegex.MatchString(segment) ||
			hexEncodedRegex.MatchString(segment) {

			templatizedSegments[i] = "{id}"
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
