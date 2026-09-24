package odigosconfiguration

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func gcOwnedBy(managedBy string, profilesHash string) map[string]string {
	labels := map[string]string{}
	if managedBy != "" {
		labels[k8sconsts.OdigosProfilesManagedByLabel] = managedBy
	}
	if profilesHash != "" {
		labels[k8sconsts.OdigosProfilesHashLabel] = profilesHash
	}
	return labels
}

func gcObjectMeta(namespace string, name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: namespace, Name: name, Labels: labels}
}

// gcSurvivors lists every object of the three kinds the profile garbage collector
// touches, across all namespaces, keyed as "Kind/namespace/name".
func gcSurvivors(t *testing.T, c client.Client) map[string]bool {
	t.Helper()

	ctx := context.Background()
	survivors := map[string]bool{}

	processors := odigosv1alpha1.ProcessorList{}
	if err := c.List(ctx, &processors); err != nil {
		t.Fatalf("list processors: %v", err)
	}
	for i := range processors.Items {
		survivors[fmt.Sprintf("Processor/%s/%s", processors.Items[i].Namespace, processors.Items[i].Name)] = true
	}

	rules := odigosv1alpha1.InstrumentationRuleList{}
	if err := c.List(ctx, &rules); err != nil {
		t.Fatalf("list instrumentation rules: %v", err)
	}
	for i := range rules.Items {
		survivors[fmt.Sprintf("InstrumentationRule/%s/%s", rules.Items[i].Namespace, rules.Items[i].Name)] = true
	}

	actions := odigosv1alpha1.ActionList{}
	if err := c.List(ctx, &actions); err != nil {
		t.Fatalf("list actions: %v", err)
	}
	for i := range actions.Items {
		survivors[fmt.Sprintf("Action/%s/%s", actions.Items[i].Namespace, actions.Items[i].Name)] = true
	}

	return survivors
}

// The profile garbage collector deletes cluster resources, and the only thing standing
// between it and a resource a user created is the odigos.io/managed-by value: a
// resource with no profiles-hash label satisfies the "hash != current" requirement, so
// "managed-by == profile" is the whole guard. This matters more since the UI started
// stamping odigos.io/managed-by=odigos-ui on the InstrumentationRules it creates.
//
// Reconciling with no effective profiles skips the apply step entirely, which leaves
// the collection pass as the only thing running.
func TestApplyProfileManifestsDeletesOnlyStaleProfileManagedResources(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, consts.DefaultOdigosNamespace)

	const odigosVersion = "v1.2.3"
	const otherNamespace = "other-namespace"

	currentHash := calculateProfilesDeploymentHash(nil, odigosVersion)
	staleHash := calculateProfilesDeploymentHash([]common.ProfileName{"size_m"}, odigosVersion)
	if currentHash == staleHash {
		t.Fatalf("the two profile lists must hash differently, got %q for both", currentHash)
	}

	ns := consts.DefaultOdigosNamespace
	objects := []client.Object{
		&odigosv1alpha1.Processor{
			ObjectMeta: gcObjectMeta(ns, "stale-profile-processor", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, staleHash)),
		},
		&odigosv1alpha1.InstrumentationRule{
			ObjectMeta: gcObjectMeta(ns, "stale-profile-rule", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, staleHash)),
		},
		&odigosv1alpha1.Action{
			ObjectMeta: gcObjectMeta(ns, "stale-profile-action", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, staleHash)),
		},
		&odigosv1alpha1.Processor{
			ObjectMeta: gcObjectMeta(ns, "current-profile-processor", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, currentHash)),
		},
		&odigosv1alpha1.InstrumentationRule{
			ObjectMeta: gcObjectMeta(ns, "current-profile-rule", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, currentHash)),
		},
		&odigosv1alpha1.Action{
			ObjectMeta: gcObjectMeta(ns, "current-profile-action", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, currentHash)),
		},
		&odigosv1alpha1.InstrumentationRule{
			ObjectMeta: gcObjectMeta(ns, "ui-rule", gcOwnedBy(k8sconsts.OdigosUIManagedByValue, "")),
		},
		&odigosv1alpha1.Action{
			ObjectMeta: gcObjectMeta(ns, "ui-action", gcOwnedBy(k8sconsts.OdigosUIManagedByValue, "")),
		},
		&odigosv1alpha1.Action{
			ObjectMeta: gcObjectMeta(ns, "interrogation-loop-action", gcOwnedBy(k8sconsts.OdigosInterrogationLoopManagedByValue, "")),
		},
		&odigosv1alpha1.InstrumentationRule{
			ObjectMeta: gcObjectMeta(ns, "yaml-rule", nil),
		},
		&odigosv1alpha1.Processor{
			ObjectMeta: gcObjectMeta(ns, "yaml-processor", nil),
		},
		// A rule that would be collected if it lived in the odigos namespace, so that
		// dropping the namespace from the list options is observable.
		&odigosv1alpha1.InstrumentationRule{
			ObjectMeta: gcObjectMeta(otherNamespace, "stale-profile-rule", gcOwnedBy(k8sconsts.OdigosProfilesManagedByValue, staleHash)),
		},
	}

	scheme := runtime.NewScheme()
	if err := odigosv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add odigos types to scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	r := &odigosConfigurationController{
		Client:        fakeClient,
		Scheme:        scheme,
		OdigosVersion: odigosVersion,
	}

	if err := r.applyProfileManifests(context.Background(), nil); err != nil {
		t.Fatalf("applyProfileManifests: %v", err)
	}

	want := map[string]bool{
		"Processor/" + ns + "/current-profile-processor":                true,
		"InstrumentationRule/" + ns + "/current-profile-rule":           true,
		"Action/" + ns + "/current-profile-action":                      true,
		"InstrumentationRule/" + ns + "/ui-rule":                        true,
		"Action/" + ns + "/ui-action":                                   true,
		"Action/" + ns + "/interrogation-loop-action":                   true,
		"InstrumentationRule/" + ns + "/yaml-rule":                      true,
		"Processor/" + ns + "/yaml-processor":                           true,
		"InstrumentationRule/" + otherNamespace + "/stale-profile-rule": true,
	}

	got := gcSurvivors(t, fakeClient)
	for key := range want {
		if !got[key] {
			t.Errorf("%s was deleted but is not managed by a profile deployment", key)
		}
	}
	for key := range got {
		if !want[key] {
			t.Errorf("%s survived but is a stale profile-managed resource", key)
		}
	}
	if len(got) != len(want) {
		t.Errorf("expected %d surviving resources, got %d: %v", len(want), len(got), gcSortedKeys(got))
	}
}

func gcSortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
