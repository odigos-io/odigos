package controllers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type OdigletPodsWebhook struct {
	client.Client
	Decoder admission.Decoder
}

var _ admission.Handler = &OdigletPodsWebhook{}

func (o *OdigletPodsWebhook) InjectDecoder(d admission.Decoder) error {
	o.Decoder = d
	return nil
}

// Handle implements the admission.Handler interface to mutate Odiglet Pod objects at creation/update time.
//
// CURRENTLY DISABLED: This webhook is currently disabled and will be used for future features.
// The infrastructure is in place to intercept and modify odiglet pods before they are created.
// When enabled in the future, this webhook will be used to:
// - Dynamically adjust resource limits based on node characteristics
// - Inject additional configuration
//
// To enable mutations, add your custom logic below and remove the early return.
func (o *OdigletPodsWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	// DISABLED: No mutations are applied in the current phase
	// The webhook is registered but passes through without modifications
	return admission.Allowed("webhook disabled - no mutations applied")

}
