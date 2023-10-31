package resources

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OdigosCloudSecretName      = "odigos-cloud-proxy"
	odigosCloudTokenEnvName    = "ODIGOS_CLOUD_TOKEN"
	odigosCloudApiKeySecretKey = "api-key"
)

func GetCurrentOdigosCloudSecret(ctx context.Context, client *kube.Client, ns string) (*corev1.Secret, error) {
	secret, err := client.CoreV1().Secrets(ns).Get(ctx, OdigosCloudSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// apparently, k8s is not setting the type meta for the object obtained with Get.
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	return secret, err
}

func IsOdigosCloud(ctx context.Context, client *kube.Client, ns string) (bool, error) {
	sec, err := GetCurrentOdigosCloudSecret(ctx, client, ns)
	if err != nil {
		return false, err
	}
	isOdigosCloud := sec != nil
	return isOdigosCloud, nil
}

func NewKeyvalSecret(ns string, apiKey string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OdigosCloudSecretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			odigosCloudApiKeySecretKey: apiKey,
		},
	}
}

type odigosCloudResourceManager struct {
	client *kube.Client
	ns     string
	config *odigosv1.OdigosConfigurationSpec
	apiKey *string
}

// the odigos cloud resource manager supports the following follows:
// 1. odigos cloud is not enabled - should not install the resource manager.
// 2. User has provided an api key - should install the resource manager and use the new apiKey as parameter to the new function.
// 3. User wishes to update resources but leave the api key as is - install resource manager and set apiKey to nil.
func NewOdigosCloudResourceManager(client *kube.Client, ns string, config *odigosv1.OdigosConfigurationSpec, apiKey *string) ResourceManager {
	return &odigosCloudResourceManager{client: client, ns: ns, config: config, apiKey: apiKey}
}

func (a *odigosCloudResourceManager) Name() string { return "Cloud" }

func (a *odigosCloudResourceManager) InstallFromScratch(ctx context.Context) error {

	if a.apiKey == nil {
		// no-op - just apply the resources again to make sure the labels are up to date.
		sec, err := GetCurrentOdigosCloudSecret(ctx, a.client, a.ns)
		if err != nil || sec == nil {
			return err
		}

		// Without the following line, I get an error:
		// ERROR metadata.managedFields must be nil
		// But not sure if this is the right way to fix it.
		sec.ManagedFields = nil
		return a.client.ApplyResources(ctx, a.config.OdigosVersion, []client.Object{sec})
	}

	resources := []client.Object{
		NewKeyvalSecret(a.ns, *a.apiKey),
	}
	return a.client.ApplyResources(ctx, a.config.OdigosVersion, resources)
}
