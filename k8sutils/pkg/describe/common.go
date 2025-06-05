package describe

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

func wrapTextInRed(text string) string {
	return "\033[31m" + text + "\033[0m"
}

func wrapTextInGreen(text string) string {
	return "\033[32m" + text + "\033[0m"
}

func wrapTextInYellow(text string) string {
	return "\033[33m" + text + "\033[0m"
}

func describeText(sb *strings.Builder, indent int, isListKey bool, printftext string, args ...interface{}) {
	indentText := strings.Repeat("  ", indent)
	if isListKey {
		listKeyIndent := indent - 1
		if listKeyIndent < 0 {
			listKeyIndent = 0
		}
		indentText = strings.Repeat("  ", listKeyIndent) + "- "
	}
	lineText := fmt.Sprintf(printftext, args...)
	fmt.Fprintf(sb, "%s%s\n", indentText, lineText)
}

func printProperty(sb *strings.Builder, indent int, property *properties.EntityProperty) {
	if property == nil {
		return
	}
	text := fmt.Sprintf("%s: %v", property.Name, property.Value)
	switch property.Status {
	case properties.PropertyStatusSuccess:
		text = wrapTextInGreen(text)
	case properties.PropertyStatusError:
		text = wrapTextInRed(text)
	case properties.PropertyStatusTransitioning:
		text = wrapTextInYellow(text)
	}

	describeText(sb, indent, property.ListKey, "%s", text)
}
