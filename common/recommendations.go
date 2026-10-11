package common

type RecommendationType string

const (
	RecommendationTypeInferDBAttributes   RecommendationType = "InferDBAttributes"
	RecommendationTypeAutoGoOffsetUpdater RecommendationType = "AutoGoOffsetUpdater"
	RecommendationTypeEnableOwnMetrics    RecommendationType = "EnableOwnMetrics"
	RecommendationTypeSampleHealthProbes  RecommendationType = "SampleHealthProbes"
	RecommendationTypeUrlTemplatization   RecommendationType = "UrlTemplatization"
)
