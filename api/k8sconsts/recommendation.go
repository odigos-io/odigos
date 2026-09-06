package k8sconsts

const (
	// RecommendationDismissedLabel marks a Recommendation CR as dismissed by the user.
	// Presence with value "true" means the recommendation is dismissed.
	RecommendationDismissedLabel = "odigos.io/recommendation-dismissed"
)
