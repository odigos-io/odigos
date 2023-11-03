package labels

const (
	OdigosSystemLabelKey       = "odigos.io/system-object"
	OdigosSystemConfigLabelKey = "odigos.io/config"
	OdigosSystemLabelValue     = "true"
)

var OdigosSystem = map[string]string{
	OdigosSystemLabelKey: OdigosSystemLabelValue,
}
