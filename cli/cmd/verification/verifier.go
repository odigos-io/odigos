package verification

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
)

type Verifier interface {
	Verify(context.Context) error
}

var _ Verifier = (*VerifierFunc)(nil)

type VerifierFunc func(context.Context) error

func (v VerifierFunc) Verify(ctx context.Context) error {
	return v(ctx)
}

func PreInstallVerifierFns(spec odigosv1.OdigosConfigurationSpec, client *kube.Client) []Verifier {
	return []Verifier{
		VerifyOPAGatekeeper(spec),
		VerifyNodeKernel(client),
	}
}
