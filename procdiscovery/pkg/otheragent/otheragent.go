// Package otheragent detects another instrumentation agent already running in a
// process, so Odigos avoids double-instrumenting it. Shared by odiglet and vm-agent.
package otheragent

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// OtherAgent is the detection result; consumers map Name to their own type.
type OtherAgent struct {
	Name string
}

// SignalType is the kind of evidence that identifies an agent in a process.
type SignalType int

const (
	EnvPresent       SignalType = iota // env var Key exists (any value)
	EnvValueContains                   // env var Key's value contains Match
	CmdlineContains                    // command line contains Match
	LibLoaded                          // a mapped file in /proc/<pid>/maps contains Match
)

// KnownAgent is one detection marker. Several entries may share a Name (one per
// loader / language). A single match means the agent is present.
type KnownAgent struct {
	Name     string
	Language common.ProgrammingLanguage // empty = any language
	Signal   SignalType
	Key      string
	Match    string
}

// Detect returns the first known agent found in the process, or nil. Entries
// with a Language are only checked when it matches lang; the rest always run.
func Detect(pcx *process.ProcessContext, lang common.ProgrammingLanguage) *OtherAgent {
	for i := range KnownAgents {
		a := &KnownAgents[i]
		if a.Language != "" && a.Language != lang {
			continue
		}
		if matches(pcx, a) {
			return &OtherAgent{Name: a.Name}
		}
	}
	return nil
}

// EnvKeysOfInterest returns the env var names referenced by env-based entries,
// so odiglet can add them to its env-collection whitelist.
func EnvKeysOfInterest() map[string]struct{} {
	keys := make(map[string]struct{})
	for i := range KnownAgents {
		if a := &KnownAgents[i]; a.Signal == EnvPresent || a.Signal == EnvValueContains {
			keys[a.Key] = struct{}{}
		}
	}
	return keys
}

func matches(pcx *process.ProcessContext, a *KnownAgent) bool {
	switch a.Signal {
	case EnvPresent:
		_, ok := envValue(pcx, a.Key)
		return ok
	case EnvValueContains:
		v, ok := envValue(pcx, a.Key)
		return ok && strings.Contains(v, a.Match)
	case CmdlineContains:
		return strings.Contains(pcx.CmdLine, a.Match)
	case LibLoaded:
		f, err := pcx.GetMapsFile()
		return err == nil && utils.IsMapsFileContainsBinary(f, []string{a.Match})
	}
	return false
}

// envValue reads an env var from the detailed set, then the overwrite set (where
// Odigos keeps values like LD_PRELOAD).
func envValue(pcx *process.ProcessContext, key string) (string, bool) {
	if v, ok := pcx.Environments.DetailedEnvs[key]; ok {
		return v, true
	}
	v, ok := pcx.Environments.OverwriteEnvs[key]
	return v, ok
}
