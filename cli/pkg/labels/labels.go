package labels

const (
	OdigosSystemLabelKey        = "odigos.io/system-object"
	OdigosSystemVersionLabelKey = "odigos.io/version"
	OdigosSystemLabelValue      = "true"
)

var OdigosSystem = map[string]string{
	OdigosSystemLabelKey: OdigosSystemLabelValue,
}
