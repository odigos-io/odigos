package urltemplate

import "strings"

// SplitPath splits a URL path into segments (without a leading empty segment).
// "/users/123" -> ["users", "123"], true; "users/123" -> ["users", "123"], false.
func SplitPath(path string) ([]string, bool) {
	hasLeadingSlash := path != "" && path[0] == '/'
	if hasLeadingSlash {
		path = path[1:]
	}
	if path == "" {
		return []string{""}, hasLeadingSlash
	}
	return strings.Split(path, "/"), hasLeadingSlash
}
