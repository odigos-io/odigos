package common

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/consts"
)

// StartPprofServer starts the pprof server. This is blocking, so it should be run in a goroutine.
// If the server is unable to start, the process will exit with a non-zero exit code.
func StartPprofServer(logger logr.Logger) {
	logger.Info("Starting pprof server")
	addr := fmt.Sprintf(":%d", consts.PprofOdigosPort)
	if err := http.ListenAndServe(addr, nil); err != nil {
		os.Exit(-1)
	}
}