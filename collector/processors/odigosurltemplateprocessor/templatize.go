package odigosurltemplateprocessor

import (
	"regexp"
	"strings"
)

var (
	uuidRegex   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	numberRegex = regexp.MustCompile(`^\d+$`)
)

func templatizeURLPath(path string) string {
	modified := false
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if uuidRegex.MatchString(segment) || numberRegex.MatchString(segment) {
			segments[i] = ":id"
			modified = true
		}
	}
	if !modified {
		return path
	} else {
		return strings.Join(segments, "/")
	}
}
