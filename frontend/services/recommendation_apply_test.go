package services

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/recommendations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// The GraphQL layer hands the recommendation type straight through as a string, so the lookup
// only succeeds for the exact spelling the catalog uses.
func TestApplyRecommendationRemediation_RejectsATypeTheCatalogDoesNotKnow(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)

	for _, recommendationType := range []string{
		"",
		"urltemplatization",
		"URL_TEMPLATIZATION",
		"url-templatization",
		"TypeTheCatalogDoesNotKnow",
	} {
		t.Run(recommendationType, func(t *testing.T) {
			err := ApplyRecommendationRemediation(context.Background(), c, recommendationType, "EnableUrlTemplatization")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found in catalog")
		})
	}
}

func TestApplyRecommendationRemediation_RejectsARemediationTheRecommendationDoesNotHave(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)

	err := ApplyRecommendationRemediation(context.Background(), recommendationsConfigClient(t),
		string(common.RecommendationTypeSampleHealthProbes), "EnableUrlTemplatization")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `remediation type "EnableUrlTemplatization" not found for recommendation "SampleHealthProbes"`)
}

// EnableOwnMetricsStore ships with apply examples but no steps, because enabling the metrics
// store is a Helm change the backend cannot make. The UI is told so through canApplyViaUi.
func TestApplyRecommendationRemediation_RejectsARemediationWithNoSteps(t *testing.T) {
	useRecommendationsNamespace(t)
	catalog := recommendationCatalogEntry(t, common.RecommendationTypeEnableOwnMetrics)
	require.Len(t, catalog.Remediations, 1, "the shipped catalog changed, update this test")
	require.Empty(t, catalog.Remediations[0].Steps, "the shipped catalog changed, update this test")
	setFakeRecommendationClient(t)

	err := ApplyRecommendationRemediation(context.Background(), recommendationsConfigClient(t),
		string(common.RecommendationTypeEnableOwnMetrics), catalog.Remediations[0].Type)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no steps")
}

func TestApplyRecommendationRemediation_EditConfigStepsCreateTheLocalUiConfigMap(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.NoError(t, err)
	cfg := storedLocalUiConfiguration(t, c)
	require.NotNil(t, cfg.Sampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling.Enabled)
	assert.True(t, *cfg.Sampling.K8sHealthProbesSampling.Enabled)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling.KeepPercentage)
	assert.Equal(t, float64(0), *cfg.Sampling.K8sHealthProbesSampling.KeepPercentage)

	// The ConfigMap is garbage collected with odigos-configuration, so it has to be created
	// owned by it rather than orphaned.
	cm := storedLocalUiConfigMap(t, c)
	require.Len(t, cm.OwnerReferences, 1)
	assert.Equal(t, consts.OdigosConfigurationName, cm.OwnerReferences[0].Name)
	assert.Equal(t, "odigos-configuration-uid", string(cm.OwnerReferences[0].UID))
	assert.Equal(t, "local-ui", cm.Labels[k8sconsts.OdigosSystemConfigLabelKey])
}

// odigos-local-ui-config holds every setting the UI owns, so applying a recommendation must
// patch only the paths its steps name.
func TestApplyRecommendationRemediation_EditConfigStepsKeepTheRestOfTheLocalUiConfig(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	dryRun := true
	c := recommendationsConfigClient(t, recommendationsLocalUiConfigMap(t, &common.OdigosConfiguration{
		McpAccessMode: common.McpAccessModeReadWrite,
		ClusterName:   "production-eu",
		Sampling:      &common.SamplingConfiguration{DryRun: &dryRun},
	}))

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.NoError(t, err)
	cfg := storedLocalUiConfiguration(t, c)
	assert.Equal(t, common.McpAccessModeReadWrite, cfg.McpAccessMode)
	assert.Equal(t, "production-eu", cfg.ClusterName)
	require.NotNil(t, cfg.Sampling)
	require.NotNil(t, cfg.Sampling.DryRun)
	assert.True(t, *cfg.Sampling.DryRun, "a sibling sampling setting must survive")
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling.Enabled)
	assert.True(t, *cfg.Sampling.K8sHealthProbesSampling.Enabled)
}

// The two health-probe remediations are mutually exclusive choices for the same config paths,
// so picking the second one has to replace the first rather than be ignored.
func TestApplyRecommendationRemediation_ASecondRemediationChoiceReplacesTheFirst(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)
	ctx := context.Background()

	require.NoError(t, ApplyRecommendationRemediation(ctx, c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes"))
	require.NoError(t, ApplyRecommendationRemediation(ctx, c,
		string(common.RecommendationTypeSampleHealthProbes), "SparseSampleHealthProbes"))

	cfg := storedLocalUiConfiguration(t, c)
	require.NotNil(t, cfg.Sampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling.KeepPercentage)
	assert.Equal(t, float64(2), *cfg.Sampling.K8sHealthProbesSampling.KeepPercentage)
}

func TestApplyRecommendationRemediation_MultipleEditConfigStepsAllLand(t *testing.T) {
	useRecommendationsNamespace(t)
	catalog := recommendationCatalogEntry(t, common.RecommendationTypeAutoGoOffsetUpdater)
	require.Len(t, catalog.Remediations[0].Steps, 2, "the shipped catalog changed, update this test")
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeAutoGoOffsetUpdater), "DirectAutoOffsets")

	require.NoError(t, err)
	cfg := storedLocalUiConfiguration(t, c)
	// The mode the catalog writes has to be one the scheduler's offsets controller recognises;
	// the two are independent hand-written sites with nothing linking them.
	assert.Equal(t, string(k8sconsts.OffsetCronJobModeDirect), cfg.GoAutoOffsetsMode)
	assert.Equal(t, "0 0 * * *", cfg.GoAutoOffsetsCron)
}

func TestApplyRecommendationRemediation_ConfigWriteFailureIsReported(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	// No odigos-configuration ConfigMap, so upsertLocalUiConfig cannot resolve an owner.
	c := fake.NewClientBuilder().WithScheme(newScheme()).Build()

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get odigos-configuration for owner reference")
}

// Two people applying recommendations at the same time collide on the one ConfigMap; the write
// is wrapped in RetryOnConflict so the second one re-reads and wins instead of failing.
func TestApplyRecommendationRemediation_RetriesAConflictingConfigWrite(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)

	updates := 0
	c := fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithObjects(
			&v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name: consts.OdigosConfigurationName, Namespace: recommendationsTestNamespace,
			}},
			recommendationsLocalUiConfigMap(t, &common.OdigosConfiguration{ClusterName: "production-eu"}),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				updates++
				if updates == 1 {
					return apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"},
						consts.OdigosLocalUiConfigName, errors.New("modified by someone else"))
				}
				return cl.Update(ctx, obj, opts...)
			},
		}).
		Build()

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.NoError(t, err)
	assert.Equal(t, 2, updates, "the conflicting write must be retried")
	cfg := storedLocalUiConfiguration(t, c)
	require.NotNil(t, cfg.Sampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling.Enabled)
	assert.True(t, *cfg.Sampling.K8sHealthProbesSampling.Enabled)
	assert.Equal(t, "production-eu", cfg.ClusterName)
}

// Only a NotFound may be read as "the ConfigMap does not exist yet". Any other read failure -
// an RBAC change, for instance - must surface, or the settings the UI already stored are
// silently replaced by a freshly created ConfigMap holding only this recommendation.
func TestApplyRecommendationRemediation_AFailedLocalUiConfigReadIsNotTreatedAsAbsent(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)

	creates := 0
	c := fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithObjects(&v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: consts.OdigosConfigurationName, Namespace: recommendationsTestNamespace,
		}}).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, cl client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if key.Name == consts.OdigosLocalUiConfigName {
					return apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, key.Name,
						errors.New("configmaps is forbidden"))
				}
				return cl.Get(ctx, key, obj, opts...)
			},
			Create: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				creates++
				return cl.Create(ctx, obj, opts...)
			},
		}).
		Build()

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err), "expected the forbidden error to surface, got %v", err)
	assert.Zero(t, creates, "a read failure must not lead to creating a replacement ConfigMap")
}

// An odigos-local-ui-config that exists but carries no data at all still has to be filled in.
func TestApplyRecommendationRemediation_FillsInAnEmptyLocalUiConfigMap(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t, &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: consts.OdigosLocalUiConfigName, Namespace: recommendationsTestNamespace,
	}})

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.NoError(t, err)
	cfg := storedLocalUiConfiguration(t, c)
	require.NotNil(t, cfg.Sampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling)
	require.NotNil(t, cfg.Sampling.K8sHealthProbesSampling.Enabled)
	assert.True(t, *cfg.Sampling.K8sHealthProbesSampling.Enabled)
}

func TestApplyRecommendationRemediation_UnparseableLocalUiConfigIsReported(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosLocalUiConfigName, Namespace: recommendationsTestNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "clusterName: [not, a, string]"},
	})

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeSampleHealthProbes), "DropAllHealthProbes")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse existing config")
}

func TestApplyRecommendationRemediation_ActionStepCreatesAUiManagedAction(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)

	err := ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeUrlTemplatization), "EnableUrlTemplatization")

	require.NoError(t, err)
	actions := storedActions(t)
	require.Len(t, actions, 1)
	created := actions[0]
	assert.Equal(t, "url-templatization", created.Name)
	assert.Equal(t, recommendationsTestNamespace, created.Namespace)
	assert.Equal(t, "URL Templatization", created.Spec.ActionName)
	require.NotNil(t, created.Spec.URLTemplatization,
		"the action must carry the empty config block, or the Action webhook rejects it")

	// The label is what keeps the action editable from the UI afterwards; assert it through the
	// production reader rather than by comparing to the same constants the writer used.
	assert.True(t, isActionUiGenerated(&created))
}

// An action-only remediation must not touch odigos-local-ui-config at all.
func TestApplyRecommendationRemediation_ActionStepDoesNotWriteTheLocalUiConfig(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)

	require.NoError(t, ApplyRecommendationRemediation(context.Background(), c,
		string(common.RecommendationTypeUrlTemplatization), "EnableUrlTemplatization"))

	var cm v1.ConfigMap
	err := c.Get(context.Background(),
		client.ObjectKey{Namespace: recommendationsTestNamespace, Name: consts.OdigosLocalUiConfigName}, &cm)
	assert.True(t, apierrors.IsNotFound(err), "expected no local UI config to be written, got %v", err)
}

// Clicking apply twice must not fail; the action already exists with the name from the catalog.
func TestApplyRecommendationRemediation_ApplyingAnActionTwiceIsIdempotent(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)
	c := recommendationsConfigClient(t)
	ctx := context.Background()

	require.NoError(t, ApplyRecommendationRemediation(ctx, c,
		string(common.RecommendationTypeInferDBAttributes), "EnableInferDbAttributes"))
	require.NoError(t, ApplyRecommendationRemediation(ctx, c,
		string(common.RecommendationTypeInferDBAttributes), "EnableInferDbAttributes"))

	assert.Len(t, storedActions(t), 1)
}

func TestApplyRecommendationRemediation_ActionCreateFailureIsReported(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	clientset := setFakeRecommendationClient(t)
	clientset.PrependReactor("create", "actions", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("actions are forbidden")
	})

	err := ApplyRecommendationRemediation(context.Background(), recommendationsConfigClient(t),
		string(common.RecommendationTypeUrlTemplatization), "EnableUrlTemplatization")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplyOdigosAction steps[0]")
	assert.Contains(t, err.Error(), `create Action "url-templatization"`)
	assert.Contains(t, err.Error(), "actions are forbidden")
}

func TestApplyOdigosActionStep_RejectsARemediationWithNoActionExample(t *testing.T) {
	useRecommendationsNamespace(t)
	setFakeRecommendationClient(t)

	err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
		Type: "SomeRemediation",
		ApplyExamples: []recommendations.ApplyExample{
			{Type: recommendations.ApplyExampleTypeHelmValues, Content: "clusterName: x\n"},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires an applyExamples entry of type")
	assert.Empty(t, storedActions(t))
}

func TestApplyOdigosActionStep_RejectsAnActionExampleItCannotUse(t *testing.T) {
	useRecommendationsNamespace(t)

	for _, tc := range []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "empty", content: "", wantErr: "content is empty"},
		{name: "only whitespace", content: "   \n\t\n", wantErr: "content is empty"},
		{name: "not yaml", content: "\tnot yaml at all", wantErr: "parse OdigosAction YAML"},
		{
			name:    "no name",
			content: "apiVersion: odigos.io/v1alpha1\nkind: Action\nspec:\n  actionName: Nameless\n",
			wantErr: "metadata.name is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setFakeRecommendationClient(t)

			err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
				Type: "SomeRemediation",
				ApplyExamples: []recommendations.ApplyExample{
					{Type: recommendations.ApplyExampleTypeOdigosAction, Content: tc.content},
				},
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Empty(t, storedActions(t), "nothing may be created when the example is unusable")
		})
	}
}

func TestApplyOdigosActionStep_KeepsLabelsTheApplyExampleDeclares(t *testing.T) {
	useRecommendationsNamespace(t)
	setFakeRecommendationClient(t)

	err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
		Type: "SomeRemediation",
		ApplyExamples: []recommendations.ApplyExample{{
			Type: recommendations.ApplyExampleTypeOdigosAction,
			Content: "apiVersion: odigos.io/v1alpha1\nkind: Action\nmetadata:\n  name: labelled-action\n" +
				"  labels:\n    app.kubernetes.io/part-of: odigos\n" +
				"spec:\n  actionName: Labelled\n  signals:\n  - TRACES\n  inferDbAttributes: {}\n",
		}},
	})

	require.NoError(t, err)
	actions := storedActions(t)
	require.Len(t, actions, 1)
	assert.Equal(t, "odigos", actions[0].Labels["app.kubernetes.io/part-of"])
	assert.True(t, isActionUiGenerated(&actions[0]))
}

// The namespace is taken from the environment, not from the apply example, so an example that
// names a different namespace still lands in the odigos namespace.
func TestApplyOdigosActionStep_OverridesTheNamespaceFromTheApplyExample(t *testing.T) {
	useRecommendationsNamespace(t)
	setFakeRecommendationClient(t)

	err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
		Type: "SomeRemediation",
		ApplyExamples: []recommendations.ApplyExample{{
			Type: recommendations.ApplyExampleTypeOdigosAction,
			Content: "apiVersion: odigos.io/v1alpha1\nkind: Action\nmetadata:\n  name: elsewhere\n" +
				"  namespace: default\nspec:\n  actionName: Elsewhere\n  inferDbAttributes: {}\n",
		}},
	})

	require.NoError(t, err)
	actions := storedActions(t)
	require.Len(t, actions, 1)
	assert.Equal(t, recommendationsTestNamespace, actions[0].Namespace)
}

func TestApplyOdigosActionStep_AnAlreadyExistingActionIsNotAnError(t *testing.T) {
	useRecommendationsNamespace(t)
	setFakeRecommendationClient(t, &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: "url-templatization", Namespace: recommendationsTestNamespace},
		Spec:       v1alpha1.ActionSpec{ActionName: "created by someone else"},
	})

	err := applyOdigosActionStep(context.Background(), recommendations.Remediation{
		Type: "SomeRemediation",
		ApplyExamples: []recommendations.ApplyExample{{
			Type: recommendations.ApplyExampleTypeOdigosAction,
			Content: "apiVersion: odigos.io/v1alpha1\nkind: Action\nmetadata:\n  name: url-templatization\n" +
				"spec:\n  actionName: URL Templatization\n  urlTemplatization: {}\n",
		}},
	})

	require.NoError(t, err)
	actions := storedActions(t)
	require.Len(t, actions, 1)
	assert.Equal(t, "created by someone else", actions[0].Spec.ActionName,
		"the existing action must be left alone")
}
