package urltemplate

import "strings"

// RulePathSegment is one "/"-separated part of a path rule.
type RulePathSegment struct {

	// if wildcard is true, it mean that this path segment always matches the path segment.
	// the content of the path will not be templated,
	// and it's the user responsibility to ensure this value has low cardinality.
	Wildcard bool

	// if this rule path segment is a static string (e.g. "users"), this value will be non-empty
	StaticString string

	// it this rule segment path is replaced with templated name, the TemplateName will be non-empty
	TemplateName string
}

// PathRule is a parsed path pattern made of RulePathSegments.
// Used for URL templatization and for matching paths/routes against a rule.
type PathRule []RulePathSegment

func parseRuleTemplateString(ruleTemplateString string) (string, error) {
	templateName := strings.TrimSpace(ruleTemplateString)
	if templateName == "" {
		templateName = "id" // default to name id if not provided
	}
	return templateName, nil
}

// ParseUserInputRuleString splits a path rule on "/" into a PathRule.
// Segments may be static ("users"), wildcard ("*"), or templated ("{name}" / "{}", defaulting to "id").
func ParseUserInputRuleString(userInputRule string) (PathRule, error) {
	segments := strings.Split(userInputRule, "/")
	if strings.HasPrefix(userInputRule, "/") {
		// if the rule starts with a /, remove it
		// this is to avoid empty string in the first segment
		segments = segments[1:]
	}

	ruleSegments := make(PathRule, len(segments))

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
			// parse the template name
			templateName, err := parseRuleTemplateString(templatizationRule)
			if err != nil {
				return nil, err
			}
			ruleSegments[i] = RulePathSegment{
				TemplateName: templateName,
			}
		} else {
			// static string segment — must match path segment exactly
			ruleSegments[i] = RulePathSegment{
				StaticString: segment,
			}
		}
	}

	return ruleSegments, nil
}
