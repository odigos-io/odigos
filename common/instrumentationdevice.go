package common

type OdigosInstrumentationDevice string

// This is the resource namespace of the lister in k8s device plugin manager.
// from the "github.com/kubevirt/device-plugin-manager" package source:
// GetResourceNamespace must return namespace (vendor ID) of implemented Lister. e.g. for
// resources in format "color.example.com/<color>" that would be "color.example.com".
const OdigosResourceNamespace = "instrumentation.odigos.io"
