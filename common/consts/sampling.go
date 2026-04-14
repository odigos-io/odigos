package consts

// Sampling category constants
type SamplingCategory string

const (
	SamplingCategoryNoise          SamplingCategory = "noise"
	SamplingCategoryHighlyRelevant SamplingCategory = "highly relevant"
	SamplingCategoryCostReduction  SamplingCategory = "cost reduction"
)
