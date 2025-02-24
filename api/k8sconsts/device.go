package k8sconsts

const (
	// the name of the device that only mounts the odigos agents root directory,
	// allowing any agent to be access it's files.
	// it would be more ideal to only mount what is needed,
	// but it's not desirable to have tons of different devices for each case.
	OdigosGenericDeviceName = "instrumentation.odigos.io/generic"
)
