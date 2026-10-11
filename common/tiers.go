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

// IsEnterprise reports whether the tier ships the enterprise images and features.
// The tier defaults to community when unset, so the check fails closed.
func (t OdigosTier) IsEnterprise() bool {
	return t != CommunityOdigosTier
}
