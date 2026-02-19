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

	// RollbackRecoveryAtAnnotation is set on a Source to request recovery from a rollback.
	// The sourceinstrumentation controller copies it to the InstrumentationConfig annotation.
	// The rollout controller compares it with RollbackRecoveryProcessedAtAnnotation to decide
	// whether recovery is needed.
	RollbackRecoveryAtAnnotation = "odigos.io/rollback-recovery"

	// RollbackRecoveryProcessedAtAnnotation is set on InstrumentationConfig by the rollout
	// controller to record the last recovery timestamp that was processed. When this matches
	// RollbackRecoveryAtAnnotation on the same IC, the recovery has been handled.
	RollbackRecoveryProcessedAtAnnotation = "odigos.io/rollback-recovery-processed"

	// This label is used to mark resources that are managed by Helm.
	AppManagedByHelmLabel = "app.kubernetes.io/managed-by"
	AppManagedByHelmValue = "Helm"
)
