package ruby

import (
	"path/filepath"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RubyInspector struct{}

var (
	processNames = []string{"ruby", "rails", "rails server", "rake", "rackup", "puma", "unicorn", "gem", "bundler", "irb", "pry"}
)

func (n *RubyInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	if utils.IsBaseExeContainsProcessName(baseExe, processNames) {
		return common.RubyProgrammingLanguage, true
	}

	return "", false
}

func (n *RubyInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *RubyInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := pcx.GetDetailedEnvsValue(process.RubyVersionConst); exists {
		return common.GetVersion(value)
	}

	// TODO: The env is not always exposed, check if we can get the version from the executable/memory
	// Use OTel "determineRubyVersion" function as reference:
	// https://github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/4377c7485ec426eb1210098593e6175d2d53bcd8/interpreter/ruby/ruby.go

	return nil
}
