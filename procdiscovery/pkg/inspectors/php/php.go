package php

import (
	"bufio"
	"bytes"
	"debug/elf"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PhpInspector struct{}

var (
	// Matches php executables: php, php-cgi, php-fpm, php82, php8.3, php-fpm82
	phpExecutableRegex = regexp.MustCompile("^php(-cgi|-fpm)?[0-9.]*$")

	// Extracts version from ELF .rodata section: "X-Powered-By: PHP/8.3.4"
	versionRegex  = regexp.MustCompile(`X-Powered-By:\s*PHP/(\d+\.\d+\.\d+)`)
	versionPrefix = "X-Powered-By: PHP/"

	// Matches .so files: libphp.so.8.3, mod_php82.so, php8.2.so
	phpSoVersionRe = regexp.MustCompile(`php([0-9.]*)\.so(?:\.(\d+)\.?(\d+)?\.?(\d+)?)?`)

	// Matches php paths: /php/8.3/, /php8.2/, /php82/
	phpPathVersionRe = regexp.MustCompile(`/php/?([0-9.]+)/`)
)

func (n *PhpInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if phpExecutableRegex.MatchString(filepath.Base(pcx.ExePath)) {
		return common.PhpProgrammingLanguage, true
	}

	return "", false
}

func (n *PhpInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	mapsFile, err := pcx.GetMapsFile()
	if err != nil {
		return "", false
	}

	scanner := bufio.NewScanner(mapsFile)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "php") {
			continue
		}
		if phpSoVersionRe.MatchString(line) || phpPathVersionRe.MatchString(line) {
			return common.PhpProgrammingLanguage, true
		}
	}
	return "", false
}

func (n *PhpInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	if value, exists := pcx.GetDetailedEnvsValue(process.PhpVersionConst); exists {
		return value
	}

	// first try to get version from the executable's .rodata section
	if vers := getVersionFromExecutable(pcx); vers != "" {
		return vers
	}

	// if that fails, try to extract version from the maps file
	// i.e. from lines containing .so files or php paths, which may include version info in their names
	if mapsFile, err := pcx.GetMapsFile(); err == nil {
		if vers := extractVersionFromMapsFile(mapsFile); vers != "" {
			return vers
		}
	}

	return ""
}

func extractVersionFromMapsFile(mapsFile process.ProcessFile) string {
	scanner := bufio.NewScanner(mapsFile)
	var bestVersion string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "php") {
			continue
		}

		var vers string

		if m := phpSoVersionRe.FindStringSubmatch(line); len(m) > 1 {
			// .so pattern: libphp.so.8.3 or mod_php82.so
			if m[1] != "" {
				vers = normalizeVersion(m[1]) // version before .so: "82" → "8.2"
			} else {
				vers = joinNonEmpty(m[2], m[3], m[4]) // version after .so: "8", "3" → "8.3"
			}
		} else if m := phpPathVersionRe.FindStringSubmatch(line); len(m) > 1 {
			// path pattern: /php/8.3/ or /php8.2/
			vers = normalizeVersion(m[1])
		}

		if vers != "" && isBetterVersion(vers, bestVersion) {
			bestVersion = vers
		}
	}

	return bestVersion
}

func joinNonEmpty(parts ...string) string {
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, ".")
}

// normalizeVersion expands concatenated versions: "82" → "8.2", "834" → "8.3.4"
func normalizeVersion(version string) string {
	if version == "" || strings.Contains(version, ".") {
		return version
	}

	switch len(version) {
	case 2:
		return string(version[0]) + "." + string(version[1])
	case 3:
		return string(version[0]) + "." + string(version[1]) + "." + string(version[2])
	default:
		return version
	}
}

func isBetterVersion(newVer, currentBest string) bool {
	if currentBest == "" {
		return true
	}
	return strings.Count(newVer, ".") > strings.Count(currentBest, ".")
}

func getVersionFromExecutable(pcx *process.ProcessContext) string {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return ""
	}

	file, err := elf.NewFile(exeFile)
	if err != nil {
		return ""
	}

	needle := []byte(versionPrefix)
	for _, section := range file.Sections {
		if section.Name != ".rodata" {
			continue
		}

		data, err := section.Data()
		if err != nil || len(data) == 0 {
			continue
		}

		idx := bytes.Index(data, needle)
		if idx < 0 {
			continue
		}
		idx += len(needle)

		zeroIdx := bytes.IndexByte(data[idx:], 0)
		if zeroIdx < 0 {
			continue
		}

		versionStr := string(data[idx : idx+zeroIdx])
		if matches := versionRegex.FindStringSubmatch(versionPrefix + versionStr); matches != nil {
			return versionStr
		}
	}

	return ""
}
