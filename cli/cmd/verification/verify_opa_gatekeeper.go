package verification

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
)

func VerifyOPAGatekeeper(spec odigosv1.OdigosConfigurationSpec) VerifierFunc {
	return func(ctx context.Context) error {
		// TODO(clavinjune): verify whether odigos's namespace is allow-listed or not
		return nil
	}
}
