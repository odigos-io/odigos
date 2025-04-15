package odigosurltemplateprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	uuidRegex   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	numberRegex = regexp.MustCompile(`^\d+$`)
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
		if uuidRegex.MatchString(segment) || numberRegex.MatchString(segment) {
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
