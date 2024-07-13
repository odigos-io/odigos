package configsections

import "github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"

type ConfigSectionName string

// This interface is the message sent to opamp client to configure aspects of the SDK
const (
	RemoteConfigSdkConfigSectionName                      ConfigSectionName = "SDK"
	RemoteConfigInstrumentationLibrariesConfigSectionName ConfigSectionName = "InstrumentationLibraries"
)

type TraceSignalGeneralConfig struct {

	// reflects if the trace signals is enabled for this SDK.
	// if false, the SDK should not produce any traces.
	// this is to spare computation on the agent in case the receiver is not setup to receive traces.
	Enabled bool `json:"enabled"`

	// by using this value, one can choose the behavior for instrumentation libraries
	// for which there is no explicit configuration.
	// one can set this value to true to allow all instrumentation libraries to produce traces, unless explicitly disabled.
	// one can set this value to false to disable all instrumentation libraries, unless explicitly enabled.
	DefaultEnabledValue bool `json:"defaultEnabledValue"`
}

type RemoteConfigSdk struct {
	RemoteResourceAttributes []configresolvers.ResourceAttribute `json:"remoteResourceAttributes"`

	// general configuration for trace signals in the SDK level.
	TraceSignal TraceSignalGeneralConfig `json:"traceSignal"`
}

type RemoteConfigInstrumentationLibrary struct {
	Name   string                                   `json:"name"`
	Traces RemoteConfigInstrumentationLibraryTraces `json:"traces"`
}

type RemoteConfigInstrumentationLibraryTraces struct {
	Enabled *bool `json:"enabled,omitempty"`
}
