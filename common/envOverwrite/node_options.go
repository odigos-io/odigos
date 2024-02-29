package envOverwrite

import (
	"fmt"
	"regexp"

	"github.com/keyval-dev/odigos/common"
)

type nodeOptions struct{}

func (n *nodeOptions) EnvName() string {
	return "NODE_OPTIONS"
}

func (n *nodeOptions) ValueFor(sdkType common.OtelSdkType) string {
	if sdkType == common.NativeOtelSdkType {
		return fmt.Sprintf("--require %s", "/var/odigos/nodejs/autoinstrumentation.js")
	}

	return fmt.Sprintf("--require %s", "/var/odigos/nodejs-ebpf/dist/loader.js")
}

func (n *nodeOptions) Revert(str string) string {
	// Define the pattern to match "--require /var/odigos/*.js"
	// Note: In the pattern, `.*` matches any sequence of characters (non-greedily)
	// and `\` is escaped as `\\`.
	pattern := "--require /var/odigos/.*?\\.js"
	// Compile the regular expression
	re, err := regexp.Compile(pattern)
	if err != nil {
		// If the pattern is invalid, return the original string
		return str
	}

	// Replace the matched patterns with an empty string
	result := re.ReplaceAllString(str, "")
	return result
}
