package common

type OdigosTier string

const (

	// community is the opensource tier of odigos
	CommunityOdigosTier OdigosTier = "community"

	// cloud is the SaaS offering of odigos
	CloudOdigosTier OdigosTier = "cloud"

	// on premises comes with enterprise features and does not require
	// network connectivity to odigos cloud
	OnPremOdigosTier OdigosTier = "onprem"
)
