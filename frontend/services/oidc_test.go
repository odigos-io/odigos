package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// the oidc block helm renders when all three ui.oidc* values are set.
const fullOidcConfig = `
oidc:
  tenantUrl: https://127.0.0.1:1
  clientId: odigos-ui
  clientSecret: secretRef:odigos-oidc
`

// the oidc block helm renders when the client secret is left empty: the block is
// still written, but templates/ui/secret.yaml then creates no odigos-oidc secret.
const oidcConfigWithoutSecret = `
oidc:
  tenantUrl: https://127.0.0.1:1
  clientId: odigos-ui
`

func effectiveConfigCM(body string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()},
		Data:       map[string]string{consts.OdigosConfigurationFileName: body},
	}
}

func oidcSecret(value string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OidcSecretName, Namespace: env.GetCurrentNamespace()},
		Data:       map[string][]byte{consts.OidcClientSecretProperty: []byte(value)},
	}
}

type failedGet struct {
	resource string
	err      error
}

func setOidcTestClient(t *testing.T, objs []runtime.Object, failures ...failedGet) {
	t.Helper()
	clientset := k8sfake.NewSimpleClientset(objs...)
	for _, f := range failures {
		err := f.err
		clientset.PrependReactor("get", f.resource, func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, err
		})
	}
	kube.SetDefaultClient(&kube.Client{Interface: clientset})
	t.Cleanup(func() { kube.SetDefaultClient(nil) })
}

// A transient API server error on the effective-config read used to reach log.Fatalf,
// which terminates the UI process. That read happens on every request handled by the
// OIDC middleware, so an API server blip (control plane upgrade, client throttling)
// took the whole UI pod down instead of failing the single request.
func TestGetOidcOauthConfig_TransientConfigMapErrorIsReturned(t *testing.T) {
	setOidcTestClient(t, nil, failedGet{"configmaps", apierrors.NewServiceUnavailable("etcd leader changed")})

	cfg, err := GetOidcOauthConfig(context.Background())
	if err == nil {
		t.Fatal("expected an error when the effective config cannot be read, got nil")
	}
	if cfg != nil {
		t.Fatalf("expected no oauth2 config on error, got %#v", cfg)
	}
	if !apierrors.IsServiceUnavailable(err) {
		t.Fatalf("expected the api error to be wrapped, got %v", err)
	}
}

// Malformed YAML in the effective config also used to reach log.Fatalf.
func TestGetOidcOauthConfig_UnparsableConfigIsReturned(t *testing.T) {
	setOidcTestClient(t, []runtime.Object{effectiveConfigCM("oidc: [unterminated")})

	if _, err := GetOidcOauthConfig(context.Background()); err == nil {
		t.Fatal("expected an error for an unparsable effective config, got nil")
	}
}

// A failed secret read must not silently disable OIDC: returning (nil, nil) here
// makes OidcMiddleware skip authentication for the request.
func TestGetOidcOauthConfig_SecretReadErrorIsReturned(t *testing.T) {
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, consts.OidcSecretName, fmt.Errorf("rbac denied"))
	setOidcTestClient(t, []runtime.Object{effectiveConfigCM(fullOidcConfig)}, failedGet{"secrets", forbidden})

	cfg, err := GetOidcOauthConfig(context.Background())
	if err == nil {
		t.Fatal("expected an error when the oidc secret cannot be read, got nil")
	}
	if cfg != nil {
		t.Fatalf("expected no oauth2 config on error, got %#v", cfg)
	}
}

// An absent secret is a supported configuration, not a failure, and must stay
// "OIDC is not configured" rather than becoming an error.
func TestGetOidcOauthConfig_MissingSecretMeansNotConfigured(t *testing.T) {
	setOidcTestClient(t, []runtime.Object{effectiveConfigCM(oidcConfigWithoutSecret)})

	cfg, err := GetOidcOauthConfig(context.Background())
	if err != nil {
		t.Fatalf("expected no error when the oidc secret does not exist, got %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected no oauth2 config when oidc is not fully configured, got %#v", cfg)
	}
}

func TestGetOidcOauthConfig_NoOidcBlockMeansNotConfigured(t *testing.T) {
	setOidcTestClient(t, []runtime.Object{effectiveConfigCM("clusterName: prod\n")})

	cfg, err := GetOidcOauthConfig(context.Background())
	if err != nil {
		t.Fatalf("expected no error when oidc is not configured, got %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected no oauth2 config when oidc is not configured, got %#v", cfg)
	}
}

// Provider discovery talks to the IdP. An unreachable IdP used to reach log.Fatalf
// through OidcAuthCallback, which is an unauthenticated route, so an IdP outage
// crash-looped the UI pod.
func TestGetOidcTokenVerifier_ProviderDiscoveryFailureIsReturned(t *testing.T) {
	setOidcTestClient(t, []runtime.Object{effectiveConfigCM(fullOidcConfig), oidcSecret("s3cret")})

	verifier, err := GetOidcTokenVerifier(context.Background())
	if err == nil {
		t.Fatal("expected an error when the oidc provider cannot be discovered, got nil")
	}
	if verifier != nil {
		t.Fatalf("expected no verifier on error, got %#v", verifier)
	}
}
