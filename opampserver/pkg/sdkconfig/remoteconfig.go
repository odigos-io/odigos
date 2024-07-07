package sdkconfig

type ConfigSectionName string

// This interface is the message sent to opamp client to configure aspects of the SDK
const (
	RemoteConfigSdkConfigSectionName                      ConfigSectionName = "SDK"
	RemoteConfigInstrumentationLibrariesConfigSectionName ConfigSectionName = "InstrumentationLibraries"
)

type ResourceAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RemoteConfigSdk struct {
	RemoteResourceAttributes []ResourceAttribute `json:"remoteResourceAttributes"`
}

type RemoteConfigInstrumentationLibrary struct {
	Name   string                                   `json:"name"`
	Traces RemoteConfigInstrumentationLibraryTraces `json:"traces"`
}

type RemoteConfigInstrumentationLibraryTraces struct {
	Disabled bool `json:"disabled,omitempty"`
}
