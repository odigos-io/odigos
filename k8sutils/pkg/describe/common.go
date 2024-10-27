package describe

import (
	"fmt"
	"strings"
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

func wrapTextSuccessOfFailure(text string, success bool) string {
	if success {
		return wrapTextInGreen(text)
	} else {
		return wrapTextInRed(text)
	}
}

func describeText(sb *strings.Builder, indent int, printftext string, args ...interface{}) {
	indentText := strings.Repeat("  ", indent)
	lineText := fmt.Sprintf(printftext, args...)
	sb.WriteString(fmt.Sprintf("%s%s\n", indentText, lineText))
}
