package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8s "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const dsecNamespace = consts.DefaultOdigosNamespace

// dsecCluster points the package-level kube client at fakes seeded with the given
// objects, and pins the namespace the services resolve.
func dsecCluster(t *testing.T, objects ...runtime.Object) (*k8sfake.Clientset, *odigosfake.Clientset) {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, dsecNamespace)

	var coreObjects, odigosObjects []runtime.Object
	for _, object := range objects {
		switch object.(type) {
		case *v1alpha1.Destination, *v1alpha1.DestinationList:
			odigosObjects = append(odigosObjects, object)
		default:
			coreObjects = append(coreObjects, object)
		}
	}

	coreClient := k8sfake.NewSimpleClientset(coreObjects...)
	// The typed fake stores a created object under its empty Name rather than
	// expanding GenerateName the way the API server does, which would make every
	// generated-name assertion below trivially equal.
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

	return coreClient, odigosClient
}

func dsecSecret(name string, data map[string]string) *k8s.Secret {
	binary := map[string][]byte{}
	for key, value := range data {
		binary[key] = []byte(value)
	}
	return &k8s.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: dsecNamespace},
		Data:       binary,
	}
}

func dsecDestination(name string, secretName string) *v1alpha1.Destination {
	dest := &v1alpha1.Destination{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: dsecNamespace, UID: "dest-uid"},
		Spec:       v1alpha1.DestinationSpec{Type: "datadog", DestinationName: "my datadog"},
	}
	if secretName != "" {
		dest.Spec.SecretRef = &k8s.LocalObjectReference{Name: secretName}
	}
	return dest
}

// dsecWrites returns the non-read actions recorded on the fake, so a test can prove
// a refusal path wrote nothing at all.
func dsecWrites(client *k8sfake.Clientset) []k8stesting.Action {
	writes := []k8stesting.Action{}
	for _, action := range client.Actions() {
		switch action.GetVerb() {
		case "get", "list", "watch":
		default:
			writes = append(writes, action)
		}
	}
	return writes
}

// ---------------------------------------------------------------------------
// GetDestinationSecretFields
// ---------------------------------------------------------------------------

func TestGetDestinationSecretFieldsReadsNothingWhenTheDestinationHasNoSecretRef(t *testing.T) {
	coreClient, _ := dsecCluster(t, dsecSecret("odigos.io.dest.datadog-12345", map[string]string{"DATADOG_API_KEY": "k"}))

	fields, err := GetDestinationSecretFields(context.Background(), dsecNamespace, dsecDestination("dest", ""))

	require.NoError(t, err)
	assert.Empty(t, fields)
	assert.NotNil(t, fields, "callers merge into this map, so it must not be nil")
	assert.Empty(t, coreClient.Actions(), "no secret should have been read")
}

func TestGetDestinationSecretFieldsDecodesEverySecretKey(t *testing.T) {
	coreClient, _ := dsecCluster(t, dsecSecret("odigos.io.dest.datadog-12345", map[string]string{
		"DATADOG_API_KEY": "api-key-value",
		"DATADOG_APP_KEY": "app-key-value",
	}))

	fields, err := GetDestinationSecretFields(context.Background(), dsecNamespace,
		dsecDestination("dest", "odigos.io.dest.datadog-12345"))

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"DATADOG_API_KEY": "api-key-value",
		"DATADOG_APP_KEY": "app-key-value",
	}, fields)

	require.Len(t, coreClient.Actions(), 1)
	getAction, ok := coreClient.Actions()[0].(k8stesting.GetAction)
	require.True(t, ok)
	assert.Equal(t, dsecNamespace, getAction.GetNamespace(), "the secret is read from the odigos namespace")
	assert.Equal(t, "odigos.io.dest.datadog-12345", getAction.GetName())
}

// A destination whose secret has been deleted out from under it fails the read
// rather than degrading to "no secret fields", which is what makes a single broken
// destination fail the whole destinations list in the UI.
func TestGetDestinationSecretFieldsPropagatesAMissingSecretAsAnError(t *testing.T) {
	dsecCluster(t)

	fields, err := GetDestinationSecretFields(context.Background(), dsecNamespace,
		dsecDestination("dest", "odigos.io.dest.datadog-12345"))

	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "got %v", err)
	assert.Nil(t, fields)
}

func TestGetDestinationSecretFieldsPropagatesAForbiddenRead(t *testing.T) {
	coreClient, _ := dsecCluster(t)
	coreClient.PrependReactor("get", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "s", errors.New("denied"))
	})

	_, err := GetDestinationSecretFields(context.Background(), dsecNamespace,
		dsecDestination("dest", "odigos.io.dest.datadog-12345"))

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err), "an RBAC denial must not read as a missing secret: %v", err)
}

// ---------------------------------------------------------------------------
// CreateDestinationSecret
// ---------------------------------------------------------------------------

func TestCreateDestinationSecretNamesTheSecretAfterTheDestinationType(t *testing.T) {
	coreClient, _ := dsecCluster(t)

	ref, err := CreateDestinationSecret(context.Background(), "datadog",
		map[string]string{"DATADOG_API_KEY": "api-key-value"}, dsecNamespace)

	require.NoError(t, err)
	require.NotNil(t, ref)
	// Spelled out rather than rebuilt from the production constant, so renaming the
	// prefix is visible here; the odiglet and the UI both match on it.
	assert.True(t, strings.HasPrefix(ref.Name, "odigos.io.dest.datadog-"), "got %q", ref.Name)

	created, err := coreClient.CoreV1().Secrets(dsecNamespace).Get(context.Background(), ref.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "odigos.io.dest.datadog-", created.GenerateName)
	assert.Equal(t, map[string]string{"DATADOG_API_KEY": "api-key-value"}, created.StringData)
	assert.Empty(t, created.OwnerReferences, "the owner reference is stamped separately, after the destination exists")
}

func TestCreateDestinationSecretWritesEverySecretFieldItWasGiven(t *testing.T) {
	coreClient, _ := dsecCluster(t)

	fields := map[string]string{"FIRST": "first-value", "SECOND": "second-value", "THIRD": "third-value"}

	ref, err := CreateDestinationSecret(context.Background(), "datadog", fields, dsecNamespace)
	require.NoError(t, err)

	created, err := coreClient.CoreV1().Secrets(dsecNamespace).Get(context.Background(), ref.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, fields, created.StringData)
}

func TestCreateDestinationSecretPropagatesTheCreateFailure(t *testing.T) {
	coreClient, _ := dsecCluster(t)
	coreClient.PrependReactor("create", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("etcd is down"))
	})

	ref, err := CreateDestinationSecret(context.Background(), "datadog",
		map[string]string{"DATADOG_API_KEY": "api-key-value"}, dsecNamespace)

	require.Error(t, err)
	assert.Nil(t, ref)
}

// ---------------------------------------------------------------------------
// AddDestinationOwnerReferenceToSecret
// ---------------------------------------------------------------------------

// The owner reference is what makes the secret disappear with its destination. All
// four of its fields are strings written one after another, so each gets its own
// assertion and its own distinguishable fixture value.
func TestAddDestinationOwnerReferenceToSecretStampsTheDestinationAsTheOwner(t *testing.T) {
	coreClient, _ := dsecCluster(t, dsecSecret("odigos.io.dest.datadog-12345", map[string]string{"K": "v"}))

	dest := dsecDestination("my-destination-object", "odigos.io.dest.datadog-12345")
	dest.UID = "11111111-2222-3333-4444-555555555555"

	require.NoError(t, AddDestinationOwnerReferenceToSecret(context.Background(), dsecNamespace, dest))

	secret, err := coreClient.CoreV1().Secrets(dsecNamespace).Get(context.Background(), "odigos.io.dest.datadog-12345", metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, secret.OwnerReferences, 1)
	assert.Equal(t, "odigos.io/v1alpha1", secret.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Destination", secret.OwnerReferences[0].Kind)
	assert.Equal(t, "my-destination-object", secret.OwnerReferences[0].Name)
	assert.EqualValues(t, "11111111-2222-3333-4444-555555555555", secret.OwnerReferences[0].UID)
}

// The patch is a JSON Patch "add" against /metadata/ownerReferences with a whole
// array as its value, which replaces the list rather than appending to it.
func TestAddDestinationOwnerReferenceToSecretReplacesOwnerReferencesThatAreAlreadyPresent(t *testing.T) {
	secret := dsecSecret("odigos.io.dest.datadog-12345", map[string]string{"K": "v"})
	secret.OwnerReferences = []metav1.OwnerReference{
		{APIVersion: "v1", Kind: "ConfigMap", Name: "someone-else", UID: "other-uid"},
	}
	coreClient, _ := dsecCluster(t, secret)

	require.NoError(t, AddDestinationOwnerReferenceToSecret(context.Background(), dsecNamespace,
		dsecDestination("my-destination-object", "odigos.io.dest.datadog-12345")))

	updated, err := coreClient.CoreV1().Secrets(dsecNamespace).Get(context.Background(), "odigos.io.dest.datadog-12345", metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, updated.OwnerReferences, 1)
	assert.Equal(t, "Destination", updated.OwnerReferences[0].Kind)
}

func TestAddDestinationOwnerReferenceToSecretPropagatesThePatchFailure(t *testing.T) {
	dsecCluster(t)

	err := AddDestinationOwnerReferenceToSecret(context.Background(), dsecNamespace,
		dsecDestination("my-destination-object", "odigos.io.dest.datadog-12345"))

	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "got %v", err)
}

// ---------------------------------------------------------------------------
// deleteDestinationAndSecret
// ---------------------------------------------------------------------------

func TestDeleteDestinationAndSecretRemovesBoth(t *testing.T) {
	coreClient, odigosClient := dsecCluster(t,
		dsecDestination("dest", "odigos.io.dest.datadog-12345"),
		dsecSecret("odigos.io.dest.datadog-12345", map[string]string{"K": "v"}),
	)

	require.NoError(t, deleteDestinationAndSecret(context.Background(), dsecDestination("dest", "odigos.io.dest.datadog-12345")))

	_, err := odigosClient.OdigosV1alpha1().Destinations(dsecNamespace).Get(context.Background(), "dest", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the destination survived: %v", err)
	_, err = coreClient.CoreV1().Secrets(dsecNamespace).Get(context.Background(), "odigos.io.dest.datadog-12345", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the secret survived: %v", err)
}

func TestDeleteDestinationAndSecretTouchesNoSecretWhenThereIsNoSecretRef(t *testing.T) {
	coreClient, odigosClient := dsecCluster(t, dsecDestination("dest", ""))

	require.NoError(t, deleteDestinationAndSecret(context.Background(), dsecDestination("dest", "")))

	_, err := odigosClient.OdigosV1alpha1().Destinations(dsecNamespace).Get(context.Background(), "dest", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))
	assert.Empty(t, dsecWrites(coreClient))
}

// If the destination delete fails the secret is deliberately left in place, so a
// transient API error cannot strip the credentials off a destination that still
// exists.
func TestDeleteDestinationAndSecretLeavesTheSecretWhenTheDestinationDeleteFails(t *testing.T) {
	coreClient, odigosClient := dsecCluster(t,
		dsecDestination("dest", "odigos.io.dest.datadog-12345"),
		dsecSecret("odigos.io.dest.datadog-12345", map[string]string{"K": "v"}),
	)
	odigosClient.PrependReactor("delete", "destinations", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("etcd is down"))
	})

	err := deleteDestinationAndSecret(context.Background(), dsecDestination("dest", "odigos.io.dest.datadog-12345"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete destination: ")
	assert.Empty(t, dsecWrites(coreClient), "the secret must not be deleted when its destination still exists")
}

func TestDeleteDestinationAndSecretReportsASecretDeleteFailureDistinctly(t *testing.T) {
	coreClient, _ := dsecCluster(t,
		dsecDestination("dest", "odigos.io.dest.datadog-12345"),
		dsecSecret("odigos.io.dest.datadog-12345", map[string]string{"K": "v"}),
	)
	coreClient.PrependReactor("delete", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("etcd is down"))
	})

	err := deleteDestinationAndSecret(context.Background(), dsecDestination("dest", "odigos.io.dest.datadog-12345"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete secret: ")
	assert.NotContains(t, err.Error(), "failed to delete destination",
		"the two failures must be distinguishable, they call for different operator action")
}

// ---------------------------------------------------------------------------
// UpdateDestination
// ---------------------------------------------------------------------------

func TestUpdateDestinationWritesTheWholeSpecToTheOdigosNamespace(t *testing.T) {
	_, odigosClient := dsecCluster(t, dsecDestination("dest", "odigos.io.dest.datadog-12345"))

	updated := dsecDestination("dest", "odigos.io.dest.datadog-12345")
	updated.Spec.DestinationName = "renamed"
	updated.Spec.Data = map[string]string{"DATADOG_SITE": "datadoghq.eu"}

	require.NoError(t, UpdateDestination(context.Background(), updated))

	stored, err := odigosClient.OdigosV1alpha1().Destinations(dsecNamespace).Get(context.Background(), "dest", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "renamed", stored.Spec.DestinationName)
	assert.Equal(t, map[string]string{"DATADOG_SITE": "datadoghq.eu"}, stored.Spec.Data)
}

func TestUpdateDestinationPropagatesTheUpdateFailure(t *testing.T) {
	_, odigosClient := dsecCluster(t, dsecDestination("dest", ""))
	odigosClient.PrependReactor("update", "destinations", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("etcd is down"))
	})

	err := UpdateDestination(context.Background(), dsecDestination("dest", ""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update destination: ")
}
