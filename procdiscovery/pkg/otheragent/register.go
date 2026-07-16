package otheragent

import "github.com/odigos-io/odigos/procdiscovery/pkg/process"

// Push the env keys we care about into the process env-collection whitelist
// (process cannot import otheragent, so we register at init).
func init() {
	process.RegisterAgentDetectionEnvKeys(EnvKeysOfInterest())
}
