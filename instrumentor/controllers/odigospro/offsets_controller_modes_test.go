package odigospro

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// offImagePrefix and offOdigosVersion are deliberately distinct from each other and from the
// image name, so the three consecutive %s in the rendered container image cannot be swapped
// without a test noticing.
const (
	offImagePrefix   = "registry.example.com/odigos"
	offOdigosVersion = "v9.8.7"
	offCron          = "17 3 * * 6"
)

// offEffectiveConfig is the effective-config document the offsets controller reads.
func offEffectiveConfig(ns string, cron string, mode k8sconsts.OffsetCronJobMode) *corev1.ConfigMap {
	return configMap(ns, consts.OdigosEffectiveConfigName, &common.OdigosConfiguration{
		ImagePrefix:       offImagePrefix,
		GoAutoOffsetsCron: cron,
		GoAutoOffsetsMode: string(mode),
	})
}

func offCronJobMeta(ns string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OffsetCronJobName}
}

// cloudSecret makes getCurrentOdigosTier report the cloud tier.
func cloudSecret(ns string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosProSecretName},
		Data:       map[string][]byte{k8sconsts.OdigosCloudApiKeySecretKey: []byte("api-key")},
	}
}

func newOffsetsReconciler(objs ...client.Object) *odigosproOffsetsController {
	r := newReconciler(objs...)
	r.OdigosVersion = offOdigosVersion
	return r
}

// offWriteRecorder counts the writes a reconcile performs, so a refusal can be proven to have
// written nothing rather than merely to have written nothing visible.
type offWriteRecorder struct {
	writes int
}

func newOffsetsReconcilerWithInterceptor(funcs interceptor.Funcs, objs ...client.Object) *odigosproOffsetsController {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return &odigosproOffsetsController{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).
			WithInterceptorFuncs(funcs).Build(),
		OdigosVersion: offOdigosVersion,
	}
}

func newRecordingOffsetsReconciler(recorder *offWriteRecorder, objs ...client.Object) *odigosproOffsetsController {
	count := func() { recorder.writes++ }
	return newOffsetsReconcilerWithInterceptor(interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			count()
			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			count()
			return c.Update(ctx, obj, opts...)
		},
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			count()
			return c.Patch(ctx, obj, patch, opts...)
		},
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			count()
			return c.Delete(ctx, obj, opts...)
		},
	}, objs...)
}

func getInitialJob(t *testing.T, c client.Client, ns string) (*batchv1.Job, bool) {
	t.Helper()
	job := &batchv1.Job{}
	err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: k8sconsts.OffsetInitialJobName}, job)
	if apierrors.IsNotFound(err) {
		return nil, false
	}
	require.NoError(t, err)
	return job, true
}

// ****************
// mode -> image + args
// ****************

// The mode picks both the image and the arguments. They are selected in the same switch, so a
// cross-wired case ships the plain CLI image with the --from-file arguments (the job then fails
// looking for an offsets file that is not in that image) or the offsets image with no
// --from-file at all (the job silently goes back to reaching out to the network).
func TestOffsetsControllerModeSelectsBothTheImageAndTheArguments(t *testing.T) {
	ns := env.GetCurrentNamespace()

	tests := []struct {
		name string
		mode k8sconsts.OffsetCronJobMode
		want corev1.Container
	}{
		{
			name: "direct fetches the offsets over the network",
			mode: k8sconsts.OffsetCronJobModeDirect,
			want: corev1.Container{
				Name:  k8sconsts.CliImageName,
				Image: offImagePrefix + "/" + k8sconsts.CliImageName + ":" + offOdigosVersion,
				Args:  []string{"pro", "update-offsets"},
			},
		},
		{
			name: "image reads the offsets baked into the image",
			mode: k8sconsts.OffsetCronJobModeImage,
			want: corev1.Container{
				Name:  k8sconsts.CliOffsetsImageName,
				Image: offImagePrefix + "/" + k8sconsts.CliOffsetsImageName + ":" + offOdigosVersion,
				Args:  []string{"pro", "update-offsets", "--from-file", "/odigos/offset_results_min.json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newOffsetsReconciler(offEffectiveConfig(ns, offCron, tt.mode), onPremSecret(ns))

			_, err := r.Reconcile(context.Background(), ctrl.Request{})
			require.NoError(t, err)

			cronJob, found := getCronJob(t, r.Client, ns)
			require.True(t, found, "mode %q must create the offsets CronJob", tt.mode)
			require.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
			assert.Equal(t, tt.want, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0])
		})
	}

	// the two modes must not agree on anything the switch chooses, or the table above could
	// pass with the cases cross-wired
	assert.NotEqual(t, tests[0].want, tests[1].want)
}

// An unset mode is the common case for an enterprise install whose cron was set from the UI
// without touching the mode, and it has to keep working like the documented default.
func TestOffsetsControllerDefaultsToDirectWhenTheModeIsUnset(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newOffsetsReconciler(offEffectiveConfig(ns, offCron, ""), onPremSecret(ns))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	cronJob, found := getCronJob(t, r.Client, ns)
	require.True(t, found)
	require.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, []string{"pro", "update-offsets"}, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
}

// The mode is free text in the effective config (the UI and central both write it), so a typo
// has to be reported rather than silently fall through the switch and ship a CronJob with an
// empty image.
func TestOffsetsControllerRejectsAnUnknownMode(t *testing.T) {
	ns := env.GetCurrentNamespace()

	recorder := &offWriteRecorder{}
	r := newRecordingOffsetsReconciler(recorder, offEffectiveConfig(ns, offCron, "dircet"), onPremSecret(ns))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid go-auto-offsets-mode: dircet")
	for _, valid := range []k8sconsts.OffsetCronJobMode{
		k8sconsts.OffsetCronJobModeDirect,
		k8sconsts.OffsetCronJobModeImage,
		k8sconsts.OffsetCronJobModeOff,
	} {
		assert.ErrorContains(t, err, string(valid), "the error must list every accepted mode")
	}
	assert.Zero(t, recorder.writes, "a rejected mode must not touch the cluster")
}

// ****************
// the enable/disable gate
// ****************

// Clearing the cron is the other way the UI turns the feature off; it must remove the CronJob
// just like setting the mode to "off" does.
func TestOffsetsControllerRemovesTheCronJobWhenTheCronIsCleared(t *testing.T) {
	ns := env.GetCurrentNamespace()

	existing := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OffsetCronJobName},
		Spec:       batchv1.CronJobSpec{Schedule: offCron},
	}
	r := newOffsetsReconciler(
		offEffectiveConfig(ns, "", k8sconsts.OffsetCronJobModeDirect),
		onPremSecret(ns),
		existing,
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	_, found := getCronJob(t, r.Client, ns)
	assert.False(t, found)
}

// Nothing to delete is the steady state on a community install, and the controller is woken by
// every odigos-deployment and pro-secret event, so this path runs constantly.
func TestOffsetsControllerDisablingIsANoOpWhenNoCronJobExists(t *testing.T) {
	ns := env.GetCurrentNamespace()

	recorder := &offWriteRecorder{}
	r := newRecordingOffsetsReconciler(recorder, offEffectiveConfig(ns, offCron, k8sconsts.OffsetCronJobModeOff), onPremSecret(ns))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)
	assert.Zero(t, recorder.writes)
}

// Custom offsets are an enterprise feature. The refusal has to be a refusal: on a community
// install the controller may not create the CronJob, the initial Job, or anything else.
func TestOffsetsControllerRefusesToRunOnTheCommunityTier(t *testing.T) {
	ns := env.GetCurrentNamespace()

	recorder := &offWriteRecorder{}
	// no odigos-pro secret at all, which is what a community install looks like
	r := newRecordingOffsetsReconciler(recorder, offEffectiveConfig(ns, offCron, k8sconsts.OffsetCronJobModeDirect))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err, "a community install is not an error, it is just not entitled")

	_, cronFound := getCronJob(t, r.Client, ns)
	assert.False(t, cronFound)
	_, jobFound := getInitialJob(t, r.Client, ns)
	assert.False(t, jobFound)
	assert.Zero(t, recorder.writes)
}

// The complementary half of the gate: both paying tiers must get through it. Testing only the
// on-prem tier would let a gate written as `tier != OnPrem` pass.
func TestOffsetsControllerRunsForEveryPayingTier(t *testing.T) {
	ns := env.GetCurrentNamespace()

	for name, secret := range map[string]*corev1.Secret{
		"on-prem": onPremSecret(ns),
		"cloud":   cloudSecret(ns),
	} {
		t.Run(name, func(t *testing.T) {
			r := newOffsetsReconciler(offEffectiveConfig(ns, offCron, k8sconsts.OffsetCronJobModeDirect), secret)

			_, err := r.Reconcile(context.Background(), ctrl.Request{})
			require.NoError(t, err)

			_, found := getCronJob(t, r.Client, ns)
			assert.True(t, found, "the %s tier is entitled to custom offsets", name)
		})
	}
}

// ****************
// getCurrentOdigosTier
// ****************

// The tier is derived from which key the odigos-pro secret happens to carry. An absent secret
// and a secret with an unrecognised key are both "community"; collapsing the two with the
// error return would turn an RBAC denial into a silent downgrade.
func TestGetCurrentOdigosTier(t *testing.T) {
	ns := env.GetCurrentNamespace()

	tests := []struct {
		name   string
		secret *corev1.Secret
		want   common.OdigosTier
	}{
		{
			name: "no odigos-pro secret",
			want: common.CommunityOdigosTier,
		},
		{
			name:   "the secret carries an on-prem token",
			secret: onPremSecret(ns),
			want:   common.OnPremOdigosTier,
		},
		{
			name:   "the secret carries a cloud api key",
			secret: cloudSecret(ns),
			want:   common.CloudOdigosTier,
		},
		{
			name: "the cloud api key wins when the secret carries both",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosProSecretName},
				Data: map[string][]byte{
					k8sconsts.OdigosCloudApiKeySecretKey: []byte("api-key"),
					k8sconsts.OdigosOnpremTokenSecretKey: []byte("token"),
				},
			},
			want: common.CloudOdigosTier,
		},
		{
			name: "the secret exists but carries neither key",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosProSecretName},
				Data:       map[string][]byte{"odigos-onprem-tokens": []byte("token")},
			},
			want: common.CommunityOdigosTier,
		},
		{
			name: "an empty token value is still an on-prem secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosProSecretName},
				Data:       map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: {}},
			},
			want: common.OnPremOdigosTier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []client.Object{}
			if tt.secret != nil {
				objs = append(objs, tt.secret)
			}
			r := newOffsetsReconciler(objs...)

			tier, err := getCurrentOdigosTier(context.Background(), r.Client, ns)

			require.NoError(t, err)
			assert.Equal(t, tt.want, tier)
		})
	}
}

// ****************
// applyCronJob
// ****************

// The initial Job exists so a freshly set cron does not wait until the next scheduled run
// before the offsets are refreshed. It has to be the same workload the CronJob schedules,
// otherwise the first run differs from every later one.
func TestOffsetsControllerTriggersAnInitialJobMatchingTheCronJob(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newOffsetsReconciler(offEffectiveConfig(ns, offCron, k8sconsts.OffsetCronJobModeImage), onPremSecret(ns))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	cronJob, found := getCronJob(t, r.Client, ns)
	require.True(t, found)
	job, found := getInitialJob(t, r.Client, ns)
	require.True(t, found, "the initial offsets Job must be created alongside the CronJob")

	assert.Equal(t, cronJob.Spec.JobTemplate.Spec.Template.Spec, job.Spec.Template.Spec)
	assert.Equal(t, k8sconsts.OffsetUpdaterServiceAccountName, job.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, corev1.RestartPolicyNever, job.Spec.Template.Spec.RestartPolicy)
	// both objects are cleaned up by `odigos uninstall` through this label
	assert.Equal(t, k8sconsts.OdigosSystemLabelValue, cronJob.Labels[k8sconsts.OdigosSystemLabelKey])
	assert.Equal(t, k8sconsts.OdigosSystemLabelValue, job.Labels[k8sconsts.OdigosSystemLabelKey])
	assert.Equal(t, offCron, cronJob.Spec.Schedule)
}

// The controller is woken by every odigos-deployment, effective-config and pro-secret event,
// so the second reconcile onwards always finds the initial Job already there. Treating that as
// an error would make the controller report a permanent failure on a healthy cluster.
func TestOffsetsControllerToleratesAnAlreadyTriggeredInitialJob(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newOffsetsReconciler(offEffectiveConfig(ns, offCron, k8sconsts.OffsetCronJobModeDirect), onPremSecret(ns))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)
	_, found := getInitialJob(t, r.Client, ns)
	require.True(t, found)

	_, err = r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err, "re-reconciling must not fail on the existing initial Job")
}
