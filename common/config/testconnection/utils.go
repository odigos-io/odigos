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

// replacePlaceholders recursively substitutes ${KEY} placeholders in string values.
// Keys may be unscoped field names (legacy / dynamic) or destination-scoped
// SecretEnvVarName values; both resolve against the flat fields map keyed by field name.
func replacePlaceholders(gmap config.GenericMap, fields map[string]string, destID string) {
	for key, value := range gmap {
		switch v := value.(type) {
		case string:
			for _, match := range placeholderRegexp.FindAllStringSubmatch(v, -1) {
				placeholderKey := match[1]
				replacement, ok := fields[placeholderKey]
				if !ok {
					for fieldName, fieldVal := range fields {
						if config.SecretEnvVarName(fieldName, destID) == placeholderKey {
							replacement, ok = fieldVal, true
							break
						}
					}
				}
				if ok {
					v = strings.ReplaceAll(v, match[0], replacement)
					gmap[key] = v
				}
			}
		case config.GenericMap:
			replacePlaceholders(v, fields, destID)
		case map[string]any:
			replacePlaceholders(v, fields, destID)
		}
	}
}
