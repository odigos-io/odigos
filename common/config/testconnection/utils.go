package testconnection

import (
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common/config"
)

var placeholderRegexp = regexp.MustCompile(`\$\{([^}]+)\}`)

// normalizeMap deep-converts a GenericMap to plain map[string]any, which confmap's decoder hooks
// require (they type-assert on map[string]any, not on named types like config.GenericMap).
func normalizeMap(gmap config.GenericMap) map[string]any {
	out := make(map[string]any, len(gmap))
	for key, value := range gmap {
		switch val := value.(type) {
		case config.GenericMap:
			out[key] = normalizeMap(val)
		case map[string]any:
			out[key] = normalizeMap(val)
		case map[string]string:
			m := make(map[string]any, len(val))
			for mk, mv := range val {
				m[mk] = mv
			}
			out[key] = m
		default:
			out[key] = value
		}
	}
	return out
}

// replacePlaceholders recursively substitutes ${KEY} placeholders in string values with fields[KEY].
func replacePlaceholders(gmap config.GenericMap, fields map[string]string) {
	for key, value := range gmap {
		switch v := value.(type) {
		case string:
			for _, match := range placeholderRegexp.FindAllStringSubmatch(v, -1) {
				if replacement, ok := fields[match[1]]; ok {
					v = strings.ReplaceAll(v, match[0], replacement)
					gmap[key] = v
				}
			}
		case config.GenericMap:
			replacePlaceholders(v, fields)
		case map[string]any:
			replacePlaceholders(v, fields)
		}
	}
}
