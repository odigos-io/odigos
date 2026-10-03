package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/recommendations"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// mbFakeActionClient is the action-shaped counterpart of useFakeRuleClient: it points
// the package-level kube client at a fake odigos clientset seeded with the given
// actions and pins the namespace the services resolve.
func mbFakeActionClient(t *testing.T, actions ...*v1alpha1.Action) {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, consts.DefaultOdigosNamespace)

	objects := make([]runtime.Object, 0, len(actions))
	for _, action := range actions {
		objects = append(objects, action)
	}

	clientset := odigosfake.NewSimpleClientset(objects...)
	previousClient := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: clientset.OdigosV1alpha1()})
	t.Cleanup(func() {
		kube.SetDefaultClient(previousClient)
	})
}

func mbStoredAction(t *testing.T, id string) *v1alpha1.Action {
	t.Helper()

	action, err := kube.DefaultClient.OdigosClient.Actions(consts.DefaultOdigosNamespace).
		Get(context.Background(), id, metav1.GetOptions{})
	require.NoError(t, err)
	return action
}

func TestCreateActionStampsManagedByOdigosUI(t *testing.T) {
	ctx := context.Background()
	name := "ui action"
	overwrite := true

	mbFakeActionClient(t)

	created, err := CreateAction(ctx, model.ActionInput{
		Type:   model.ActionTypeAddClusterInfo,
		Name:   &name,
		Fields: &model.ActionFieldsInput{OverwriteExistingValues: &overwrite},
	})
	require.NoError(t, err)
	require.Equal(t, model.ManagedByOdigosUI, created.ManagedBy)
	require.True(t, created.UIGenerated)

	stored := mbStoredAction(t, created.ID)
	require.Equal(t, k8sconsts.OdigosUIManagedByValue, stored.Labels[k8sconsts.OdigosProfilesManagedByLabel])
	require.NotContains(t, stored.Labels, k8sconsts.OdigosProfilesHashLabel,
		"a UI action must not look like it belongs to a profile deployment")
}

// Updating an action through the UI must not claim ownership of it. An action the
// profile reconciler owns keeps both its managed-by value and its profiles-hash, and
// an action applied from YAML stays unlabeled rather than becoming UI-generated.
func TestUpdateActionPreservesManagedBy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		labels map[string]string
		want   model.ManagedBy
	}{
		{
			name: "profile managed",
			labels: map[string]string{
				k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosProfilesManagedByValue,
				k8sconsts.OdigosProfilesHashLabel:      "8f14e45fceea167a",
			},
			want: model.ManagedByProfile,
		},
		{
			name:   "ui managed",
			labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue},
			want:   model.ManagedByOdigosUI,
		},
		{
			name:   "interrogation loop managed",
			labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosInterrogationLoopManagedByValue},
			want:   model.ManagedByInterrogationLoop,
		},
		{
			name:   "applied from yaml",
			labels: nil,
			want:   model.ManagedByUnknown,
		},
		{
			name:   "unrecognized owner",
			labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "helm"},
			want:   model.ManagedByUnknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			const actionID = "action-under-update"
			renamedTo := "renamed by the ui"
			overwrite := true

			mbFakeActionClient(t, &v1alpha1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      actionID,
					Namespace: consts.DefaultOdigosNamespace,
					Labels:    tc.labels,
				},
				Spec: v1alpha1.ActionSpec{ActionName: "original"},
			})

			updated, err := UpdateAction(ctx, actionID, model.ActionInput{
				Type:   model.ActionTypeAddClusterInfo,
				Name:   &renamedTo,
				Fields: &model.ActionFieldsInput{OverwriteExistingValues: &overwrite},
			})
			require.NoError(t, err)
			require.Equal(t, renamedTo, *updated.Name, "the update must have actually happened")
			require.Equal(t, tc.want, updated.ManagedBy)
			require.Equal(t, tc.want == model.ManagedByOdigosUI, updated.UIGenerated)

			require.Equal(t, tc.labels, mbStoredAction(t, actionID).Labels)
		})
	}
}

// An action applied from a recommendation must come out editable in the UI, which is
// what the odigos-ui owner means, and it must not drop the labels its manifest carries.
func TestApplyOdigosActionStepStampsManagedByOdigosUI(t *testing.T) {
	const actionName = "infer-db-attributes"

	for _, tc := range []struct {
		name           string
		manifestLabels string
	}{
		{name: "manifest without labels"},
		{name: "manifest with its own labels", manifestLabels: "  labels:\n    example.com/origin: catalog\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mbFakeActionClient(t)

			content := "apiVersion: odigos.io/v1alpha1\n" +
				"kind: Action\n" +
				"metadata:\n" +
				"  name: " + actionName + "\n" +
				tc.manifestLabels +
				"spec:\n" +
				"  actionName: Infer DB Attributes\n" +
				"  signals:\n" +
				"    - TRACES\n" +
				"  inferDbAttributes: {}\n"

			err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
				ApplyExamples: []recommendations.ApplyExample{{
					Type:    recommendations.ApplyExampleTypeOdigosAction,
					Content: content,
				}},
			})
			require.NoError(t, err)

			stored := mbStoredAction(t, actionName)
			require.Equal(t, k8sconsts.OdigosUIManagedByValue, stored.Labels[k8sconsts.OdigosProfilesManagedByLabel])
			if tc.manifestLabels != "" {
				require.Equal(t, "catalog", stored.Labels["example.com/origin"],
					"stamping the owner must not replace the manifest's own labels")
			}

			converted, err := convertActionToModel(stored)
			require.NoError(t, err)
			require.Equal(t, model.ManagedByOdigosUI, converted.ManagedBy)
			require.True(t, converted.UIGenerated)
		})
	}
}

// Re-applying a remediation over an action that already exists is a no-op, so an
// action someone applied from YAML is not adopted by the UI behind their back.
func TestApplyOdigosActionStepDoesNotAdoptExistingAction(t *testing.T) {
	const actionName = "infer-db-attributes"

	mbFakeActionClient(t, &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.ActionSpec{ActionName: "applied by hand"},
	})

	err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
		ApplyExamples: []recommendations.ApplyExample{{
			Type: recommendations.ApplyExampleTypeOdigosAction,
			Content: "apiVersion: odigos.io/v1alpha1\nkind: Action\nmetadata:\n  name: " + actionName +
				"\nspec:\n  actionName: Infer DB Attributes\n  inferDbAttributes: {}\n",
		}},
	})
	require.NoError(t, err)

	stored := mbStoredAction(t, actionName)
	require.Equal(t, "applied by hand", stored.Spec.ActionName)
	require.NotContains(t, stored.Labels, k8sconsts.OdigosProfilesManagedByLabel)

	converted, err := convertActionToModel(stored)
	require.NoError(t, err)
	require.Equal(t, model.ManagedByUnknown, converted.ManagedBy)
	require.False(t, converted.UIGenerated)
}

// uiGenerated is kept for clients that predate the managedBy enum, so the two must
// never disagree: exactly the OdigosUi owner is UI-generated.
func TestActionUIGeneratedAgreesWithManagedBy(t *testing.T) {
	cases := map[string]map[string]string{
		"<no labels>": nil,
		"helm":        {k8sconsts.OdigosProfilesManagedByLabel: "helm"},
	}
	for value := range mbRecognizedValues {
		cases[value] = map[string]string{k8sconsts.OdigosProfilesManagedByLabel: value}
	}

	uiGeneratedCount := 0
	for name, labels := range cases {
		t.Run(name, func(t *testing.T) {
			action := &v1alpha1.Action{
				ObjectMeta: metav1.ObjectMeta{Name: "action", Labels: labels},
			}

			converted, err := convertActionToModel(action)
			require.NoError(t, err)
			require.Equal(t, converted.ManagedBy == model.ManagedByOdigosUI, converted.UIGenerated)
			require.Equal(t, converted.UIGenerated, isActionUiGenerated(action))

			if converted.UIGenerated {
				uiGeneratedCount++
			}
		})
	}
	require.Equal(t, 1, uiGeneratedCount, "exactly one owner value may report uiGenerated")
}
