package verification

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
)

type Verifier interface {
	Verify(context.Context) error
}

var _ Verifier = (*VerifierFunc)(nil)

type VerifierFunc func(context.Context) error

func (v VerifierFunc) Verify(ctx context.Context) error {
	return v(ctx)
}

func PreInstallVerifierFn(spec odigosv1.OdigosConfigurationSpec) []Verifier {
	return []Verifier{
		VerifyOPAGatekeeper(spec),
	}
}
