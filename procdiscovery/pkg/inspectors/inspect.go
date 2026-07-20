package inspectors

import (
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/common/otheragent"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// InspectionResult is what one pass over a process yields: its language and any
// foreign instrumentation agents detected in it (a process can carry more than one).
type InspectionResult struct {
	Language    common.ProgramLanguageDetails
	OtherAgents []otheragent.OtherAgent
}

// Inspect runs language and foreign-agent detection against a process using one
// shared ProcessContext, so odiglet and vm-agent answer detection identically
// without re-opening /proc handles. Shared entry point for both.
func Inspect(proc process.Details) (InspectionResult, error) {
	logger := commonlogger.LoggerCompat().With("subsystem", "langdetect")
	pcx := process.NewProcessContext(proc)
	defer func() {
		if err := pcx.CloseFiles(); err != nil {
			logger.Error("Error closing files", "err", err)
		}
	}()

	lang, err := detectLanguageInContext(pcx, logger)
	// Detect regardless of language outcome; lang only scopes language-specific entries.
	agents := otheragent.DetectAll(pcx, lang.Language)

	return InspectionResult{Language: lang, OtherAgents: agents}, err
}
