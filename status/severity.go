package status

// OdigosSeverity describes how close a status reason is to the desired outcome.
// Aspects considered when choosing a value:
//   - Is in desired state (yes / no)
//   - Permanence (transient / permanent)
//   - User action items (yes / no)
//   - Severity order when multiple conditions are aggregated
//
// Values align with DesiredStateProgress in the GraphQL API.
type OdigosSeverity string

const (
	// OdigosSeverityError is used when odigos failed to determine the current or desired
	// state due to an internal error. The actual status is unknown.
	OdigosSeverityError OdigosSeverity = "Error"

	// OdigosSeverityFailure is used when not in the desired state, permanently, and the
	// user should take action (investigate, disable, etc.). Always shown if present.
	// Example: exception in agent code.
	OdigosSeverityFailure OdigosSeverity = "Failure"

	// OdigosSeverityNotice is used when not in the desired state, permanently, and the
	// user should take action (rollout / investigate). Shown first unless there is a Failure.
	// Example: manual rollout required to apply instrumentation.
	OdigosSeverityNotice OdigosSeverity = "Notice"

	// OdigosSeverityPending is used when not in the desired state, for a long-lived
	// transient reason, with no user action required. Indicates progress that may take a while.
	// Example: waiting for a cronjob to trigger runtime detection.
	OdigosSeverityPending OdigosSeverity = "Pending"

	// OdigosSeverityWaiting is used when not in the desired state, for a short-lived
	// transient reason, with no user action required. Expected to resolve shortly.
	// Example: waiting for the collectors pipeline to be deployed and ready.
	OdigosSeverityWaiting OdigosSeverity = "Waiting"

	// OdigosSeverityUnsupported is used when not in the desired state, permanently, with
	// no user action beyond being informed. Shown after transient states.
	// Example: workload language unknown, or no agent for the detected language/runtime.
	OdigosSeverityUnsupported OdigosSeverity = "Unsupported"

	// OdigosSeverityDisabled is used when the desired state is intentionally disabled by
	// user settings and that is OK. Permanent, no action required beyond being informed.
	// Example: user manually disabled instrumentation for this workload.
	OdigosSeverityDisabled OdigosSeverity = "Disabled"

	// OdigosSeveritySuccess is used when in the desired state permanently, with no user
	// action required. Shown only when nothing more severe is present.
	// Example: detection OK, agents injected, agents healthy, telemetry observed.
	OdigosSeveritySuccess OdigosSeverity = "Success"

	// OdigosSeverityIrrelevant means the condition is not applicable in the current
	// context yet. Ignored for severity aggregation.
	// Example: agent enabled is not relevant until runtime inspection is complete.
	OdigosSeverityIrrelevant OdigosSeverity = "Irrelevant"

	// OdigosSeverityUnknown is used when the severity cannot be classified.
	OdigosSeverityUnknown OdigosSeverity = "Unknown"
)
