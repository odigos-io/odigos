package testconnection

import (
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common/config"
)

// replacePlaceholders replaces placeholder values in the given GenericMap with values from the fields map.
// It traverses the GenericMap recursively and processes each string value as a template.
// If a string value contains placeholders in the format {KEY}, it replaces them with corresponding values from the fields map.
// The function supports nested GenericMaps and map[string]interface{} structures.
func replacePlaceholders(gmap config.GenericMap, fields map[string]string) {
	// Regular expression to match the ${KEY} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	for key, value := range gmap {
		switch v := value.(type) {
		case string:
			// Find all matches of the pattern ${KEY}
			matches := re.FindAllStringSubmatch(v, -1)
			for _, match := range matches {
				if len(match) == 2 {
					// match[0] is the entire match (${KEY}), match[1] is the key (KEY)
					extractedKey := match[1]
					if replacement, ok := fields[extractedKey]; ok {
						// Replace only the ${KEY} part in the original string
						v = strings.Replace(v, match[0], replacement, -1)
						// Update the map with the new value
						gmap[key] = v
					}
				}
			}
		case config.GenericMap:
			replacePlaceholders(v, fields)
		case map[string]interface{}:
			replacePlaceholders(v, fields)
		default:
			// If the value is not a string or a map, we leave it as it is
		}
	}
}