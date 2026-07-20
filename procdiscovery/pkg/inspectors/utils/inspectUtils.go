package utils

import (
	"bufio"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// versionSuffixRe matches a trailing runtime version suffix such as "3.3", "3.4.4" or "8.2",
// so a versioned interpreter binary like "ruby3.3" or "python3.11" can be matched to its
// base process name ("ruby", "python").
var versionSuffixRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

func IsProcessEqualProcessNames(pcx *process.ProcessContext, processNames []string) bool {
	baseExe := filepath.Base(pcx.ExePath)

	return slices.Contains(processNames, baseExe)
}

// IsProcessEqualProcessNamesWithVersion behaves like IsProcessEqualProcessNames but also matches
// binaries whose name carries a trailing version suffix (e.g. "ruby3.3" matches "ruby",
// "python3.11" matches "python"). Distro-packaged interpreters are commonly installed under a
// versioned name, which an exact match misses.
func IsProcessEqualProcessNamesWithVersion(pcx *process.ProcessContext, processNames []string) bool {
	baseExe := filepath.Base(pcx.ExePath)
	if slices.Contains(processNames, baseExe) {
		return true
	}
	for _, name := range processNames {
		if suffix, ok := strings.CutPrefix(baseExe, name); ok && suffix != "" && versionSuffixRe.MatchString(suffix) {
			return true
		}
	}
	return false
}

func IsMapsFileContainsBinary(mapsFile process.ProcessFile, binaries []string) bool {
	scanner := bufio.NewScanner(mapsFile)

	for scanner.Scan() {
		line := scanner.Text()

		for _, binary := range binaries {
			if strings.Contains(line, binary) {
				return true
			}
		}
	}

	return false
}
