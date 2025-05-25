package services

const (
	ActionTypeK8sAttributes        = "K8sAttributesResolver"
	ActionTypeAddClusterInfo       = "AddClusterInfo"
	ActionTypeDeleteAttribute      = "DeleteAttribute"
	ActionTypeRenameAttribute      = "RenameAttribute"
	ActionTypePiiMasking           = "PiiMasking"
	ActionTypeErrorSampler         = "ErrorSampler"
	ActionTypeLatencySampler       = "LatencySampler"
	ActionTypeProbabilisticSampler = "ProbabilisticSampler"
	ActionTypeServiceNameSampler   = "ServiceNameSampler"
)
