package k8sconsts

const (
	// OdigosInjectInstrumentationLabel is the label used to enable the mutating webhook.
	OdigosInjectInstrumentationLabel = "odigos.io/inject-instrumentation"
	// OdigosCollectorRoleLabel is the label used to identify the role of the Odigos collector.
	OdigosCollectorRoleLabel = "odigos.io/collector-role"

	// used to label resources created by profiles with the hash that created them.
	// when a new profiles is reconciled, we will apply them with a new hash
	// and use the label to identify the resources that needs to be deleted.
	OdigosProfilesHashLabel = "odigos.io/profiles-hash"

	// this label is used to mark resources that are managed by a profile.
	// when reconciling profiles, we can use this label to know which profiles needs to be deleted.
	OdigosProfilesManagedByLabel = "odigos.io/managed-by"
	OdigosProfilesManagedByValue = "profile"

	// for resources auto created by a profile, this annotation will record
	// the name of the profile that created them.
	OdigosProfileAnnotation = "odigos.io/profile"
)
