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

func describeText(sb *strings.Builder, indent int, printftext string, args ...interface{}) {
	indentText := strings.Repeat("  ", indent)
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

	describeText(sb, indent, "%s", text)
}
