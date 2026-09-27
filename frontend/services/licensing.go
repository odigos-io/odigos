package services

import "os"

const MarketplaceLicenseProvider = "aws-marketplace"

func CurrentLicenseProvider() string {
	if os.Getenv("ODIGOS_LICENSE_PROVIDER") == MarketplaceLicenseProvider {
		return MarketplaceLicenseProvider
	}
	return "odigos"
}
