package nginx

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type NginxInspector struct{}

const (
	NginxProcessName  = "nginx"
	NginxVersionRegex = `nginx/(\d+\.\d+\.\d+)`
)

func (j *NginxInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	p := pcx.Details
	if filepath.Base(p.ExePath) == NginxProcessName {
		return common.NginxProgrammingLanguage, true
	}

	return "", false
}

func (j *NginxInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (j *NginxInspector) GetRuntimeVersion(pcx *process.ProcessContext) string {
	return ""
}
