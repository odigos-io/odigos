// Package otheragent detects another instrumentation agent already running in a
// process, so Odigos avoids double-instrumenting it. It is process-source
// agnostic: callers pass a Process implementation.
package otheragent

import (
	"bufio"
	"io"
	"strings"

	"github.com/odigos-io/odigos/common"
)

type OtherAgent struct {
	Name string
}

// Process is the minimal view of a running process the detector needs.
// procdiscovery's process.ProcessContext and vm-agent's process types both
// satisfy it, keeping this package free of any /proc coupling.
type Process interface {
	// Cmdline returns the process command line.
	Cmdline() string
	// LookupEnv returns the value of an environment variable of the process.
	LookupEnv(key string) (string, bool)
	// MapsReader returns a reader over the process memory maps
	// (/proc/<pid>/maps on Linux). Only used for library-load detection.
	MapsReader() (io.Reader, error)
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

// knownAgentsByLanguage buckets KnownAgents by Language ("" = language-agnostic),
// so detection scans only the entries relevant to the process's language plus the
// agnostic ones, instead of iterating the whole table for every process.
var knownAgentsByLanguage = indexKnownAgentsByLanguage()

func indexKnownAgentsByLanguage() map[common.ProgrammingLanguage][]*KnownAgent {
	m := make(map[common.ProgrammingLanguage][]*KnownAgent)
	for i := range KnownAgents {
		agent := &KnownAgents[i]
		m[agent.Language] = append(m[agent.Language], agent)
	}
	return m
}

// DetectAll returns every distinct agent detected in the process (deduplicated by
// Name). Entries scoped to a language are only checked when it matches lang; the
// language-agnostic entries always run.
func DetectAll(p Process, lang common.ProgrammingLanguage) []OtherAgent {
	var detected []OtherAgent
	seen := make(map[string]struct{})
	scan := func(candidates []*KnownAgent) {
		for _, agent := range candidates {
			if _, done := seen[agent.Name]; done {
				continue
			}
			if agentMatchesProcess(p, agent) {
				seen[agent.Name] = struct{}{}
				detected = append(detected, OtherAgent{Name: agent.Name})
			}
		}
	}
	scan(knownAgentsByLanguage[lang])
	if lang != "" {
		scan(knownAgentsByLanguage[""]) // language-agnostic entries
	}
	return detected
}

// Detect returns the first agent detected in the process, or nil. Kept for
// callers that model a single agent (e.g. odiglet's RuntimeDetails CRD).
func Detect(p Process, lang common.ProgrammingLanguage) *OtherAgent {
	if all := DetectAll(p, lang); len(all) > 0 {
		return &all[0]
	}
	return nil
}

// Blocks reports whether detected agents should stop Odigos from instrumenting:
// at least one foreign agent is present and running concurrent agents is off.
func Blocks(detected []OtherAgent, allowConcurrentAgents bool) bool {
	return len(detected) > 0 && !allowConcurrentAgents
}

func agentMatchesProcess(p Process, agent *KnownAgent) bool {
	switch agent.Signal {
	case EnvPresent:
		_, ok := p.LookupEnv(agent.Key)
		return ok
	case EnvValueContains:
		v, ok := p.LookupEnv(agent.Key)
		return ok && strings.Contains(v, agent.Match)
	case CmdlineContains:
		return strings.Contains(p.Cmdline(), agent.Match)
	case LibLoaded:
		r, err := p.MapsReader()
		return err == nil && mapsContains(r, agent.Match)
	}
	return false
}

// mapsContains reports whether any line of a process maps reader contains name.
func mapsContains(r io.Reader, name string) bool {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), name) {
			return true
		}
	}
	return false
}
