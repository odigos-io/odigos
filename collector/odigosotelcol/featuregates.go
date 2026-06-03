package main

import (
	"fmt"
	"os"

	"go.opentelemetry.io/collector/featuregate"
)

// profilesSupportGate is the core-collector feature gate that allows a profiles
// signal pipeline to be configured. It is alpha (off by default) and, like all
// feature gates, is only settable via the --feature-gates CLI flag — i.e. via
// process argv, which cannot be changed without restarting the collector.
//
// The Odigos collector always wants profiles support available so that adding or
// removing a profiles pipeline is a pure config change (applied by a SIGHUP
// reload) with no collector restart. Rather than carry the --feature-gates flag
// in the systemd unit / EnvironmentFile (which would still require a restart to
// take effect, and only on the agent that wrote it), we enable the gate here,
// compiled into the binary. The global feature-gate registry is process-wide and
// read on every pipeline graph build (startup and every reload), so a config-only
// reload can add the profiles pipeline without a restart.
//
// This lives in its own file (not the builder-generated main.go) so it survives
// `make genodigoscol` regeneration.
const profilesSupportGate = "service.profilesSupport"

func init() {
	if err := featuregate.GlobalRegistry().Set(profilesSupportGate, true); err != nil {
		// Best-effort. If the gate has graduated to stable / been removed in a
		// future collector version, profiles support is on by default and this is
		// a no-op — log to stderr rather than crash so the (unexpected) error stays
		// visible without taking the collector down.
		fmt.Fprintf(os.Stderr, "odigos: could not enable %q feature gate: %v\n", profilesSupportGate, err)
	}
}
