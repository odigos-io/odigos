package k8sconsts

type OdigosInstrumentationDevice string

const (
	// This is the resource namespace of the lister in k8s device plugin manager.
	// from the "github.com/kubevirt/device-plugin-manager" package source:
	// GetResourceNamespace must return namespace (vendor ID) of implemented Lister. e.g. for
	// resources in format "color.example.com/<color>" that would be "color.example.com".
	OdigosResourceNamespace = "instrumentation.odigos.io"

	OdigosGenericDeviceName = "generic"

	// the name of the device that only mounts the odigos agents root directory,
	// allowing any agent to be access it's files.
	// it would be more ideal to only mount what is needed,
	// but it's not desirable to have tons of different devices for each case.
	OdigosGenericDeviceNameFull = OdigosResourceNamespace + "/" + OdigosGenericDeviceName
)
