package odigospro

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func configMap(ns, name string, cfg *common.OdigosConfiguration) *corev1.ConfigMap {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Data:       map[string]string{consts.OdigosConfigurationFileName: string(data)},
	}
}

// onPremSecret makes getCurrentOdigosTier report an enterprise tier, which the cron job requires.
func onPremSecret(ns string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosProSecretName},
		Data:       map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("token")},
	}
}

func newReconciler(objs ...client.Object) *odigosproOffsetsController {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return &odigosproOffsetsController{
		Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		OdigosVersion: "v1.0.0",
	}
}

func getCronJob(t *testing.T, c client.Client, ns string) (*batchv1.CronJob, bool) {
	t.Helper()
	cronJob := &batchv1.CronJob{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OffsetCronJobName}, cronJob)
	if apierrors.IsNotFound(err) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("unexpected error getting cron job: %v", err)
	}
	return cronJob, true
}

// The Go auto-offsets cron/mode are declared UI-settable in config/data/advanced.yaml
// (isHelmOnly: false). The UI writes them to odigos-local-ui-config and the scheduler folds that
// overlay into effective-config, so a controller that reads the helm-managed odigos-configuration
// never sees them.
func TestOffsetsControllerReadsEffectiveConfig(t *testing.T) {
	ns := env.GetCurrentNamespace()

	// helm values left the feature off; the cron was set from the UI, so it only exists in
	// effective-config.
	helmConfig := &common.OdigosConfiguration{
		ImagePrefix:       "registry.odigos.io",
		GoAutoOffsetsMode: string(k8sconsts.OffsetCronJobModeOff),
	}
	effectiveConfig := &common.OdigosConfiguration{
		ImagePrefix:       "registry.odigos.io",
		GoAutoOffsetsCron: "0 2 * * *",
		GoAutoOffsetsMode: string(k8sconsts.OffsetCronJobModeDirect),
	}

	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, helmConfig),
		configMap(ns, consts.OdigosEffectiveConfigName, effectiveConfig),
		onPremSecret(ns),
	)

	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cronJob, found := getCronJob(t, r.Client, ns)
	if !found {
		t.Fatal("expected the go offsets CronJob to be created for a cron set via the UI overlay, but none exists")
	}
	if cronJob.Spec.Schedule != effectiveConfig.GoAutoOffsetsCron {
		t.Errorf("expected schedule %q, got %q", effectiveConfig.GoAutoOffsetsCron, cronJob.Spec.Schedule)
	}
}

// The inverse direction: turning the feature off from the UI must remove the cron job, otherwise
// the `pro update-offsets` job keeps running on schedule after the user disabled it.
func TestOffsetsControllerRemovesCronJobWhenDisabledInEffectiveConfig(t *testing.T) {
	ns := env.GetCurrentNamespace()

	helmConfig := &common.OdigosConfiguration{
		ImagePrefix:       "registry.odigos.io",
		GoAutoOffsetsCron: "0 2 * * *",
		GoAutoOffsetsMode: string(k8sconsts.OffsetCronJobModeDirect),
	}
	effectiveConfig := &common.OdigosConfiguration{
		ImagePrefix:       "registry.odigos.io",
		GoAutoOffsetsCron: "0 2 * * *",
		GoAutoOffsetsMode: string(k8sconsts.OffsetCronJobModeOff),
	}
	existing := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OffsetCronJobName},
		Spec:       batchv1.CronJobSpec{Schedule: "0 2 * * *"},
	}

	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, helmConfig),
		configMap(ns, consts.OdigosEffectiveConfigName, effectiveConfig),
		onPremSecret(ns),
		existing,
	)

	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	if _, found := getCronJob(t, r.Client, ns); found {
		t.Fatal("expected the go offsets CronJob to be deleted after the mode was set to off via the UI overlay, but it still exists")
	}
}

// A missing effective-config is expected while the scheduler is starting up: the sibling
// odigosconfiguration controller creates it. Requeue instead of failing the reconcile.
func TestOffsetsControllerRequeuesWhenEffectiveConfigMissing(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, &common.OdigosConfiguration{}),
		onPremSecret(ns),
	)

	res, err := r.Reconcile(context.Background(), ctrl.Request{})
	if err != nil {
		t.Fatalf("expected no error when effective-config is missing, got %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Error("expected the reconcile to be requeued while effective-config does not exist yet")
	}
}
