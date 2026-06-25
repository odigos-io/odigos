package browser

import (
	"path/filepath"
	"slices"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// BrowserInspector heuristically identifies a container that serves a front-end web application
// (static assets / a single-page app) by recognizing common static web servers.
//
// IMPORTANT: this inspector is intentionally NOT registered in the active inspector registry
// (procdiscovery/pkg/inspectors/langdetect.go). Browser instrumentation cannot be reliably
// auto-detected from the in-pod process: a static server such as nginx or `serve` is
// indistinguishable from a backend that happens to use the same server, and a Node-based static
// server would otherwise be auto-instrumented as server-side JavaScript. Enabling this inspector by
// default would risk double-instrumenting or mis-instrumenting workloads.
//
// Browser instrumentation is therefore opt-in (see the Source containerOverride mechanism). This
// inspector is kept here as ready scaffolding for a future, explicitly gated auto-detection
// behavior, and to centralize the heuristic in one place.
type BrowserInspector struct{}

// staticServerProcessNames are executables commonly used to serve front-end assets. The list is
// deliberately conservative.
var staticServerProcessNames = []string{
	"serve",       // npm "serve"
	"http-server", // npm "http-server"
	"caddy",       // caddy file server
}

func (b *BrowserInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.Details.ExePath)
	if slices.Contains(staticServerProcessNames, baseExe) {
		return common.BrowserProgrammingLanguage, true
	}
	return "", false
}

func (b *BrowserInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (b *BrowserInspector) GetRuntimeVersion(pcx *process.ProcessContext) string {
	return ""
}
