package server

type ConfigSectionName string

// This interface is the message sent to opamp client to configure aspects of the SDK
const RemoteConfigSdkConfigSectionName ConfigSectionName = "SDK"

type RemoteConfigSdk struct {
	RemoteResourceAttributes []ResourceAttribute `json:"remoteResourceAttributes"`
}
