package k8sconsts

const (
	// OdigosAgentsMetaHashLabel is used to label pods being instrumented.
	// It can be used to count the number of instrumented pods for a workload and whether they are up to date
	// with the expected agents.
	OdigosAgentsMetaHashLabel = "odigos.io/agents-meta-hash"

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

	// RollbackRecoveryAtAnnotation is set on InstrumentationConfig to record the last
	// spec timestamp that was processed for rollback recovery. When this matches
	// IC.Spec.RollbackRecoveryAt, the recovery has been handled.
	RollbackRecoveryAtAnnotation = "odigos.io/rollback-recovery"

	// This label is used to mark resources that are managed by Helm.
	AppManagedByHelmLabel = "app.kubernetes.io/managed-by"
	AppManagedByHelmValue = "Helm"
)
