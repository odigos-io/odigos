package odigospartialk8sattrsprocessor

import (
	"fmt"
	"strings"
)

func extractServiceNameWithSuffix(fullName string) (string, error) {
	hyphenIndex := strings.LastIndex(fullName, "-")
	if hyphenIndex == -1 {
		return "", fmt.Errorf("service name '%s' does not contain a hyphen", fullName)
	}
	return fullName[:hyphenIndex], nil
}
