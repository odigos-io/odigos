package graph

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8s "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// The destination mutations route every submitted field into either the Destination
// CR or a Secret, according to the embedded catalog. These tests drive that split
// end to end against fake clients.

const dslNamespace = consts.DefaultOdigosNamespace

var dslLoadOnce sync.Once

func dslCluster(t *testing.T, objects ...runtime.Object) (*k8sfake.Clientset, *odigosfake.Clientset, MutationResolver) {
	t.Helper()

	dslLoadOnce.Do(func() { _ = destinations.Load() })
	require.NotEmpty(t, destinations.Get(), "the embedded destination catalog failed to load")

	t.Setenv(consts.CurrentNamespaceEnvVar, dslNamespace)

	var coreObjects, odigosObjects []runtime.Object
	for _, object := range objects {
		switch object.(type) {
		case *v1alpha1.Destination:
			odigosObjects = append(odigosObjects, object)
		default:
			coreObjects = append(coreObjects, object)
		}
	}

	coreClient := k8sfake.NewSimpleClientset(coreObjects...)
	// The typed fake does not expand GenerateName the way the API server does, so
	// without this every generated secret would be stored under the empty name.
	coreClient.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		secret, ok := action.(k8stesting.CreateAction).GetObject().(*k8s.Secret)
		if !ok || secret.Name != "" || secret.GenerateName == "" {
			return false, nil, nil
		}
		secret.Name = secret.GenerateName + "12345"
		return false, nil, nil
	})

	odigosClient := odigosfake.NewSimpleClientset(odigosObjects...)

	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{Interface: coreClient, OdigosClient: odigosClient.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	return coreClient, odigosClient, (&Resolver{}).Mutation()
}

// dslDatadogInput is a valid submission for the datadog catalog entry, whose two
// required fields are split across the secret boundary: DATADOG_API_KEY is declared
// `secret: true`, DATADOG_SITE is not.
func dslDatadogInput(apiKey string) model.DestinationInput {
	fields := []*model.FieldInput{{Key: "DATADOG_SITE", Value: "datadoghq.com"}}
	if apiKey != "" {
		fields = append(fields, &model.FieldInput{Key: "DATADOG_API_KEY", Value: apiKey})
	}
	return model.DestinationInput{
		Name:            "my datadog",
		Type:            "datadog",
		ExportedSignals: &model.ExportedSignalsInput{Traces: true},
		Fields:          fields,
	}
}

func dslStoredDestination(t *testing.T, odigosClient *odigosfake.Clientset, name string) *v1alpha1.Destination {
	t.Helper()
	dest, err := odigosClient.OdigosV1alpha1().Destinations(dslNamespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	return dest
}

func dslSecretNames(t *testing.T, coreClient *k8sfake.Clientset) []string {
	t.Helper()
	list, err := coreClient.CoreV1().Secrets(dslNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	names := make([]string, 0, len(list.Items))
	for _, secret := range list.Items {
		names = append(names, secret.Name)
	}
	return names
}

// ---------------------------------------------------------------------------
// CreateNewDestination
// ---------------------------------------------------------------------------

func TestCreateNewDestinationKeepsSecretFieldsOutOfTheDestinationSpec(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t)

	created, err := mutation.CreateNewDestination(context.Background(), dslDatadogInput("super-secret-key"))
	require.NoError(t, err)
	require.NotNil(t, created)

	stored := dslStoredDestination(t, odigosClient, created.ID)
	assert.Equal(t, map[string]string{"DATADOG_SITE": "datadoghq.com"}, stored.Spec.Data,
		"the api key must never be written into the Destination CR")
	// Spelled out: the destination object and its secret share this prefix, and
	// odigos matches on it elsewhere.
	assert.Equal(t, "odigos.io.dest.datadog-", stored.GenerateName)
	require.NotNil(t, stored.Spec.SecretRef)

	secret, err := coreClient.CoreV1().Secrets(dslNamespace).Get(context.Background(), stored.Spec.SecretRef.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"DATADOG_API_KEY": "super-secret-key"}, secret.StringData)

	// The owner reference is what garbage-collects the secret with its destination.
	require.Len(t, secret.OwnerReferences, 1)
	assert.Equal(t, "Destination", secret.OwnerReferences[0].Kind)
	assert.Equal(t, stored.Name, secret.OwnerReferences[0].Name)
}

func TestCreateNewDestinationCreatesNoSecretWhenNoSubmittedFieldIsSecret(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t)

	// jaeger declares three fields and none of them are secret.
	created, err := mutation.CreateNewDestination(context.Background(), model.DestinationInput{
		Name:            "my jaeger",
		Type:            "jaeger",
		ExportedSignals: &model.ExportedSignalsInput{Traces: true},
		Fields:          []*model.FieldInput{{Key: "JAEGER_URL", Value: "jaeger.default:4317"}},
	})
	require.NoError(t, err)

	stored := dslStoredDestination(t, odigosClient, created.ID)
	assert.Nil(t, stored.Spec.SecretRef)
	assert.Empty(t, dslSecretNames(t, coreClient))
}

func TestCreateNewDestinationRejectsAnUnknownDestinationType(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t)

	_, err := mutation.CreateNewDestination(context.Background(), model.DestinationInput{
		Name:            "mystery",
		Type:            "not-a-thing",
		ExportedSignals: &model.ExportedSignalsInput{Traces: true},
	})

	require.EqualError(t, err, "destination type not-a-thing not found")
	assert.Empty(t, dslSecretNames(t, coreClient))
	list, err := odigosClient.OdigosV1alpha1().Destinations(dslNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

// Validation runs before anything is written, so a rejected submission must leave
// no secret behind either.
func TestCreateNewDestinationWritesNothingWhenARequiredFieldIsMissing(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t)

	_, err := mutation.CreateNewDestination(context.Background(), dslDatadogInput(""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid destination data scheme")
	assert.Contains(t, err.Error(), "field DATADOG_API_KEY is required")
	assert.Empty(t, dslSecretNames(t, coreClient))
	list, err := odigosClient.OdigosV1alpha1().Destinations(dslNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestCreateNewDestinationRejectsAFieldTheCatalogDoesNotDeclare(t *testing.T) {
	coreClient, _, mutation := dslCluster(t)

	input := dslDatadogInput("super-secret-key")
	input.Fields = append(input.Fields, &model.FieldInput{Key: "DATADOG_TYPO", Value: "x"})

	_, err := mutation.CreateNewDestination(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "field DATADOG_TYPO is not found in config for destination type 'datadog'")
	assert.Empty(t, dslSecretNames(t, coreClient))
}

// KNOWN BUG, pinned so a fix is visible rather than silent.
//
// When the Destination create fails after its secret was already created, the
// rollback deletes a secret named after destination.Name -- the free-text display
// name the user typed -- instead of the generated secret the resolver just created.
// Two consequences, both asserted below: the credential-bearing secret is leaked
// with no owner reference to garbage-collect it, and an unrelated secret in the
// odigos namespace is deleted if its name happens to match what the user typed.
//
// Replacing `destName` with `secretRef.Name` at destinations.resolvers.go:113 fixes
// both and turns this test red.
func TestCreateNewDestinationRollbackDeletesTheWrongSecretWhenTheDestinationCreateFails(t *testing.T) {
	bystander := &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my datadog", Namespace: dslNamespace},
		Data:       map[string][]byte{"UNRELATED": []byte("value")},
	}
	coreClient, odigosClient, mutation := dslCluster(t, bystander)
	odigosClient.PrependReactor("create", "destinations", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("etcd is down"))
	})
	require.Equal(t, []string{"my datadog"}, dslSecretNames(t, coreClient), "the bystander must exist before the rollback runs")

	_, err := mutation.CreateNewDestination(context.Background(), dslDatadogInput("super-secret-key"))
	require.Error(t, err)

	remaining := dslSecretNames(t, coreClient)
	assert.Contains(t, remaining, "odigos.io.dest.datadog-12345",
		"the generated secret is leaked: the rollback does not target it")
	assert.NotContains(t, remaining, "my datadog",
		"the rollback deletes the secret named after the destination's display name instead")
}

// ---------------------------------------------------------------------------
// UpdateDestination: the four-way secret branch
// ---------------------------------------------------------------------------

func dslExistingDatadog(secretName string) *v1alpha1.Destination {
	dest := &v1alpha1.Destination{
		ObjectMeta: metav1.ObjectMeta{Name: "dest-1", Namespace: dslNamespace, UID: "dest-uid"},
		Spec: v1alpha1.DestinationSpec{
			Type:            "datadog",
			DestinationName: "my datadog",
			Data:            map[string]string{"DATADOG_SITE": "datadoghq.com"},
		},
	}
	if secretName != "" {
		dest.Spec.SecretRef = &k8s.LocalObjectReference{Name: secretName}
	}
	return dest
}

func TestUpdateDestinationCreatesTheSecretWhenOneIsAddedByTheUpdate(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t, dslExistingDatadog(""))

	_, err := mutation.UpdateDestination(context.Background(), "dest-1", dslDatadogInput("brand-new-key"))
	require.NoError(t, err)

	stored := dslStoredDestination(t, odigosClient, "dest-1")
	require.NotNil(t, stored.Spec.SecretRef)
	assert.True(t, strings.HasPrefix(stored.Spec.SecretRef.Name, "odigos.io.dest.datadog-"))

	secret, err := coreClient.CoreV1().Secrets(dslNamespace).Get(context.Background(), stored.Spec.SecretRef.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"DATADOG_API_KEY": "brand-new-key"}, secret.StringData)
	require.Len(t, secret.OwnerReferences, 1, "a secret created on update needs the same owner reference as one created on create")
	assert.Equal(t, "dest-1", secret.OwnerReferences[0].Name)
}

func TestUpdateDestinationDeletesTheSecretWhenTheLastSecretFieldIsCleared(t *testing.T) {
	existingSecret := &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-old", Namespace: dslNamespace},
		Data:       map[string][]byte{"DATADOG_API_KEY": []byte("old-key")},
	}
	coreClient, odigosClient, mutation := dslCluster(t, dslExistingDatadog("odigos.io.dest.datadog-old"), existingSecret)

	// datadog's only secret field is required, so clearing the last secret is only
	// reachable for a type whose secret field is optional: qryn-oss declares
	// QRYN_OSS_PASSWORD as `secret: true` without marking it required.
	input := model.DestinationInput{
		Name:            "my qryn",
		Type:            "qryn-oss",
		ExportedSignals: &model.ExportedSignalsInput{Traces: true},
		Fields: []*model.FieldInput{
			{Key: "QRYN_OSS_URL", Value: "http://qryn.local:3100"},
			{Key: "QRYN_OSS_PASSWORD", Value: ""},
		},
	}

	_, err := mutation.UpdateDestination(context.Background(), "dest-1", input)
	require.NoError(t, err)

	stored := dslStoredDestination(t, odigosClient, "dest-1")
	assert.Nil(t, stored.Spec.SecretRef, "the destination must stop referencing a secret that no longer exists")
	assert.Empty(t, dslSecretNames(t, coreClient))
}

func TestUpdateDestinationRewritesTheExistingSecretInPlace(t *testing.T) {
	existingSecret := &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-old", Namespace: dslNamespace},
		Data:       map[string][]byte{"DATADOG_API_KEY": []byte("old-key")},
	}
	coreClient, odigosClient, mutation := dslCluster(t, dslExistingDatadog("odigos.io.dest.datadog-old"), existingSecret)

	input := dslDatadogInput("rotated-key")
	// A site the stored destination does not already carry, so "never write the
	// data fields" is distinguishable from "write them unchanged".
	require.Equal(t, "DATADOG_SITE", input.Fields[0].Key)
	input.Fields[0].Value = "datadoghq.eu"

	_, err := mutation.UpdateDestination(context.Background(), "dest-1", input)
	require.NoError(t, err)

	stored := dslStoredDestination(t, odigosClient, "dest-1")
	require.NotNil(t, stored.Spec.SecretRef)
	assert.Equal(t, "odigos.io.dest.datadog-old", stored.Spec.SecretRef.Name, "the secret must be reused, not replaced")
	assert.Equal(t, []string{"odigos.io.dest.datadog-old"}, dslSecretNames(t, coreClient))
	assert.Equal(t, map[string]string{"DATADOG_SITE": "datadoghq.eu"}, stored.Spec.Data,
		"the non-secret fields are persisted on the destination itself")

	secret, err := coreClient.CoreV1().Secrets(dslNamespace).Get(context.Background(), "odigos.io.dest.datadog-old", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"DATADOG_API_KEY": "rotated-key"}, secret.StringData)
}

func TestUpdateDestinationTouchesNoSecretWhenThereWereNoneBeforeOrAfter(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t, &v1alpha1.Destination{
		ObjectMeta: metav1.ObjectMeta{Name: "dest-1", Namespace: dslNamespace},
		Spec:       v1alpha1.DestinationSpec{Type: "jaeger", DestinationName: "my jaeger"},
	})

	_, err := mutation.UpdateDestination(context.Background(), "dest-1", model.DestinationInput{
		Name:            "my jaeger",
		Type:            "jaeger",
		ExportedSignals: &model.ExportedSignalsInput{Traces: true},
		Fields:          []*model.FieldInput{{Key: "JAEGER_URL", Value: "jaeger.default:4317"}},
	})
	require.NoError(t, err)

	assert.Nil(t, dslStoredDestination(t, odigosClient, "dest-1").Spec.SecretRef)
	for _, action := range coreClient.Actions() {
		assert.Contains(t, []string{"get", "list", "watch"}, action.GetVerb(),
			"no secret write was expected, got a %q", action.GetVerb())
	}
}

// If the secret update fails the resolver restores the previous secret and reports
// the failure, so a half-applied update cannot leave the destination pointing at a
// secret that no longer matches what the user saw.
func TestUpdateDestinationRollsTheSecretBackWhenTheSecretUpdateFails(t *testing.T) {
	existingSecret := &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-old", Namespace: dslNamespace},
		Data:       map[string][]byte{"DATADOG_API_KEY": []byte("old-key")},
	}
	coreClient, odigosClient, mutation := dslCluster(t, dslExistingDatadog("odigos.io.dest.datadog-old"), existingSecret)

	updates := 0
	coreClient.PrependReactor("update", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		updates++
		if updates == 1 {
			return true, nil, apierrors.NewInternalError(errors.New("etcd is down"))
		}
		return false, nil, nil
	})

	_, err := mutation.UpdateDestination(context.Background(), "dest-1", dslDatadogInput("rotated-key"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update secret: ")
	assert.Equal(t, 2, updates, "the rollback write must actually be issued")

	secret, err := coreClient.CoreV1().Secrets(dslNamespace).Get(context.Background(), "odigos.io.dest.datadog-old", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string][]byte{"DATADOG_API_KEY": []byte("old-key")}, secret.Data)
	assert.Empty(t, secret.StringData, "the rolled-back secret must not carry the rejected value")

	// The destination itself must not have been updated.
	stored := dslStoredDestination(t, odigosClient, "dest-1")
	assert.Equal(t, "my datadog", stored.Spec.DestinationName)
}

func TestUpdateDestinationRejectsAnInvalidSubmissionBeforeTouchingTheCluster(t *testing.T) {
	coreClient, odigosClient, mutation := dslCluster(t, dslExistingDatadog(""))

	_, err := mutation.UpdateDestination(context.Background(), "dest-1", dslDatadogInput(""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation errors: ")
	assert.Contains(t, err.Error(), "field DATADOG_API_KEY is required")
	assert.Empty(t, dslSecretNames(t, coreClient))
	assert.Equal(t, map[string]string{"DATADOG_SITE": "datadoghq.com"},
		dslStoredDestination(t, odigosClient, "dest-1").Spec.Data)
}

func TestUpdateDestinationAddsTheCurrentStreamNameWithoutDuplicatingIt(t *testing.T) {
	existing := dslExistingDatadog("")
	existing.Spec.SourceSelector = &v1alpha1.SourceSelector{DataStreams: []string{"alpha"}}
	_, odigosClient, mutation := dslCluster(t, existing)

	input := dslDatadogInput("key")
	input.CurrentStreamName = "beta"
	_, err := mutation.UpdateDestination(context.Background(), "dest-1", input)
	require.NoError(t, err)
	require.NotNil(t, dslStoredDestination(t, odigosClient, "dest-1").Spec.SourceSelector)
	assert.Equal(t, []string{"alpha", "beta"},
		dslStoredDestination(t, odigosClient, "dest-1").Spec.SourceSelector.DataStreams)

	input.CurrentStreamName = "alpha"
	_, err = mutation.UpdateDestination(context.Background(), "dest-1", input)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"},
		dslStoredDestination(t, odigosClient, "dest-1").Spec.SourceSelector.DataStreams)
}

// ---------------------------------------------------------------------------
// DeleteDestination: detach from one stream, or delete outright
// ---------------------------------------------------------------------------

func TestDeleteDestinationOnlyDetachesTheStreamWhileOtherStreamsRemain(t *testing.T) {
	existing := dslExistingDatadog("odigos.io.dest.datadog-old")
	existing.Spec.SourceSelector = &v1alpha1.SourceSelector{DataStreams: []string{"alpha", "beta"}}
	coreClient, odigosClient, mutation := dslCluster(t, existing, &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-old", Namespace: dslNamespace},
		Data:       map[string][]byte{"DATADOG_API_KEY": []byte("old-key")},
	})

	deleted, err := mutation.DeleteDestination(context.Background(), "dest-1", "alpha")
	require.NoError(t, err)
	assert.True(t, deleted)

	stored := dslStoredDestination(t, odigosClient, "dest-1")
	require.NotNil(t, stored.Spec.SourceSelector)
	assert.Equal(t, []string{"beta"}, stored.Spec.SourceSelector.DataStreams)
	assert.Equal(t, []string{"odigos.io.dest.datadog-old"}, dslSecretNames(t, coreClient),
		"a destination still used by another stream must keep its secret")
}

func TestDeleteDestinationRemovesTheDestinationAndItsSecretWithTheLastStream(t *testing.T) {
	existing := dslExistingDatadog("odigos.io.dest.datadog-old")
	existing.Spec.SourceSelector = &v1alpha1.SourceSelector{DataStreams: []string{"alpha"}}
	coreClient, odigosClient, mutation := dslCluster(t, existing, &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-old", Namespace: dslNamespace},
		Data:       map[string][]byte{"DATADOG_API_KEY": []byte("old-key")},
	})

	deleted, err := mutation.DeleteDestination(context.Background(), "dest-1", "alpha")
	require.NoError(t, err)
	assert.True(t, deleted)

	_, err = odigosClient.OdigosV1alpha1().Destinations(dslNamespace).Get(context.Background(), "dest-1", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the destination survived: %v", err)
	assert.Empty(t, dslSecretNames(t, coreClient))
}

// A destination that was never scoped to a stream has no selector at all, which the
// delete path has to read as "nothing left to keep it alive".
func TestDeleteDestinationWithoutAStreamSelectorDeletesTheDestination(t *testing.T) {
	_, odigosClient, mutation := dslCluster(t, dslExistingDatadog(""))

	deleted, err := mutation.DeleteDestination(context.Background(), "dest-1", "alpha")
	require.NoError(t, err)
	assert.True(t, deleted)

	_, err = odigosClient.OdigosV1alpha1().Destinations(dslNamespace).Get(context.Background(), "dest-1", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the destination survived: %v", err)
}

func TestDeleteDestinationReportsAMissingDestination(t *testing.T) {
	_, _, mutation := dslCluster(t)

	deleted, err := mutation.DeleteDestination(context.Background(), "dest-1", "alpha")

	require.Error(t, err)
	assert.False(t, deleted)
	assert.Contains(t, err.Error(), "failed to get destination: ")
}

// ---------------------------------------------------------------------------
// The destinations query: who gets to see the secret values
// ---------------------------------------------------------------------------

func dslEffectiveConfig(uiMode string) *k8s.ConfigMap {
	return &k8s.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: dslNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "uiMode: " + uiMode},
	}
}

// The read-only UI mode is the only thing keeping the stored credentials out of the
// destinations query response, and it is decided per request from the effective
// config. The gate fails closed: if the effective config cannot be read the
// secrets are withheld.
func TestDestinationsQueryServesSecretValuesOnlyWhenTheUIIsWritable(t *testing.T) {
	for _, tt := range []struct {
		name             string
		configMap        *k8s.ConfigMap
		wantSecretServed bool
	}{
		{name: "writable ui serves the secret", configMap: dslEffectiveConfig("default"), wantSecretServed: true},
		{name: "readonly ui withholds the secret", configMap: dslEffectiveConfig("readonly"), wantSecretServed: false},
		{name: "an unreadable effective config fails closed", configMap: nil, wantSecretServed: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			objects := []runtime.Object{
				dslExistingDatadog("odigos.io.dest.datadog-old"),
				&k8s.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-old", Namespace: dslNamespace},
					Data:       map[string][]byte{"DATADOG_API_KEY": []byte("super-secret-key")},
				},
			}
			if tt.configMap != nil {
				objects = append(objects, tt.configMap)
			}
			dslCluster(t, objects...)

			served, err := (&Resolver{}).ComputePlatform().Destinations(context.Background(), &model.ComputePlatform{})
			require.NoError(t, err)
			require.Len(t, served, 1)

			assert.Contains(t, served[0].Fields, "DATADOG_SITE", "the non-secret field is always served")
			if tt.wantSecretServed {
				assert.Contains(t, served[0].Fields, "super-secret-key")
			} else {
				assert.NotContains(t, served[0].Fields, "super-secret-key")
				assert.NotContains(t, served[0].Fields, "DATADOG_API_KEY")
			}
		})
	}
}
