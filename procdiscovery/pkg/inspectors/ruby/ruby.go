package ruby

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RubyInspector struct{}

var (
	processNames = []string{"ruby", "rails", "rails server", "rake", "rackup", "puma", "unicorn", "gem", "bundler", "irb", "pry"}

	// /usr/local/lib/ruby/3.4.0/aarch64-linux/monitor.so
	rubyPathVersionRe = regexp.MustCompile(`/ruby/(\d+\.\d+\.\d+)/`)

	// /usr/local/lib/libruby.so.3.4.4
	rubySoVersionRe = regexp.MustCompile(`libruby\.so\.(\d+\.\d+\.\d+)`)
)

func (n *RubyInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if utils.IsProcessEqualProcessNames(pcx, processNames) {
		return common.RubyProgrammingLanguage, true
	}

	return "", false
}

func (n *RubyInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *RubyInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	// first try to get the version from env var
	if value, exists := pcx.GetDetailedEnvsValue(process.RubyVersionConst); exists {
		return value
	}

	// second, try get the version from the maps file
	mapsFie, err := pcx.GetMapsFile()
	if err == nil {
		ver := n.extractVersionFromMapsFile(mapsFie)
		if ver != "" {
			return ver
		}
	}

	// TODO: The env is not always exposed, check if we can get the version from the executable/memory
	// Use OTel "determineRubyVersion" function as reference:
	// https://github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/4377c7485ec426eb1210098593e6175d2d53bcd8/interpreter/ruby/ruby.go

	return ""
}

func (n *RubyInspector) extractVersionFromMapsFile(mapsFile process.ProcessFile) string {
	nonZeroPatchVersions := make(map[string]struct{})
	zeroPatchVersion := make(map[string]struct{})
	scanner := bufio.NewScanner(mapsFile)

	record := func(ver string) {
		parts := strings.Split(ver, ".")
		if len(parts) != 3 {
			return
		}
		if parts[2] != "0" {
			nonZeroPatchVersions[ver] = struct{}{}
		} else {
			zeroPatchVersion[ver] = struct{}{}
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.Contains(line, "ruby") {
			continue
		}

		if m := rubyPathVersionRe.FindStringSubmatch(line); len(m) > 1 {
			record(m[1])
		}

		if m := rubySoVersionRe.FindStringSubmatch(line); len(m) > 1 {
			record(m[1])
		}
	}

	// Prefer any non-zero-patch version
	if len(nonZeroPatchVersions) == 1 {
		for v := range nonZeroPatchVersions {
			return v
		}
	}

	// Fall back to a zero-patch version
	if len(zeroPatchVersion) == 1 {
		for v := range zeroPatchVersion {
			return v
		}
	}

	return ""
}
