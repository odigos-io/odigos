package controllers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	sourceTestNamespace = "checkout-ns"
	sourceTestWorkload  = "checkout"
	// A regex Source carries a hash of the pattern in its workload-name label, since
	// Kubernetes labels cannot hold regex metacharacters.
	sourceTestNameHash = "5f4dcc3b5aa765d6"
)

func sourceTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	return scheme
}

func newSourceTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	return fake.NewClientBuilder().
		WithScheme(sourceTestScheme(t)).
		WithObjects(objects...).
		Build()
}

// newValidSource builds a Source that passes validation, so that each test can invalidate
// exactly one aspect of it.
func newValidSource() *v1alpha1.Source {
	return &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "checkout-deployment-source",
			Namespace: sourceTestNamespace,
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      sourceTestWorkload,
				k8sconsts.WorkloadNamespaceLabel: sourceTestNamespace,
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindDeployment),
				defaultDataStreamLabel:           "true",
			},
		},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      sourceTestWorkload,
				Namespace: sourceTestNamespace,
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
		},
	}
}

// newValidNamespaceSource builds a namespace-wide Source that passes validation.
func newValidNamespaceSource() *v1alpha1.Source {
	source := newValidSource()
	source.Name = "checkout-ns-source"
	source.Labels[k8sconsts.WorkloadNameLabel] = sourceTestNamespace
	source.Labels[k8sconsts.WorkloadKindLabel] = string(k8sconsts.WorkloadKindNamespace)
	source.Spec.Workload = k8sconsts.PodWorkload{
		Name:      sourceTestNamespace,
		Namespace: sourceTestNamespace,
		Kind:      k8sconsts.WorkloadKindNamespace,
	}
	return source
}

// newRegexSource builds a Source that matches its workload name as a regex pattern.
func newRegexSource(name, pattern string) *v1alpha1.Source {
	source := newValidSource()
	source.Name = name
	source.Spec.MatchWorkloadNameAsRegex = true
	source.Spec.Workload.Name = pattern
	source.Labels[k8sconsts.WorkloadNameLabel] = sourceTestNameHash
	return source
}

type sourceValidationError struct {
	field  string
	detail string
}

// assertSourceValidationErrors asserts that err is the apierrors.NewInvalid error the
// webhook builds, carrying exactly the expected field errors in the expected order.
// NewInvalid renders each field error as "Invalid value: <bad value>: <detail>", so the
// detail is asserted as a suffix of the cause message.
func assertSourceValidationErrors(t *testing.T, err error, want ...sourceValidationError) {
	t.Helper()

	statusErr := &apierrors.StatusError{}
	require.True(t, errors.As(err, &statusErr), "expected a StatusError, got %v", err)
	require.True(t, apierrors.IsInvalid(err), "expected an Invalid error, got %v", err)
	require.NotNil(t, statusErr.ErrStatus.Details)

	causes := statusErr.ErrStatus.Details.Causes
	require.Len(t, causes, len(want), "unexpected field errors: %v", causes)

	for i, cause := range causes {
		assert.Equal(t, want[i].field, cause.Field)
		assert.True(t, strings.HasSuffix(cause.Message, ": "+want[i].detail),
			"field error %d is %q, expected it to end with %q", i, cause.Message, ": "+want[i].detail)
	}
}

// ****************
// SourcesDefaulter
// ****************

func TestSourcesDefaulter_SetsWorkloadLabelsOnAnEmptySource(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout-deployment-source", Namespace: sourceTestNamespace},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      sourceTestWorkload,
				Namespace: sourceTestNamespace,
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
		},
	}

	require.NoError(t, defaulter.Default(context.Background(), source))

	assert.Equal(t, map[string]string{
		"odigos.io/workload-name":       sourceTestWorkload,
		"odigos.io/workload-namespace":  sourceTestNamespace,
		"odigos.io/workload-kind":       "Deployment",
		"odigos.io/data-stream-default": "true",
	}, source.Labels)
	assert.Equal(t, []string{k8sconsts.SourceInstrumentationFinalizer}, source.Finalizers)
}

func TestSourcesDefaulter_DoesNotSetTheWorkloadNameLabelForARegexSource(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Labels = nil
	source.Spec.MatchWorkloadNameAsRegex = true
	source.Spec.Workload.Name = "checkout-.*"

	require.NoError(t, defaulter.Default(context.Background(), source))

	// Kubernetes labels cannot hold regex metacharacters, so the workload-name label is
	// left for the caller to fill in with a hash of the pattern.
	assert.NotContains(t, source.Labels, k8sconsts.WorkloadNameLabel)
	assert.Equal(t, sourceTestNamespace, source.Labels[k8sconsts.WorkloadNamespaceLabel])
	assert.Equal(t, "Deployment", source.Labels[k8sconsts.WorkloadKindLabel])
}

func TestSourcesDefaulter_KeepsLabelsThatAreAlreadySet(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Labels = map[string]string{
		k8sconsts.WorkloadNameLabel:      "already-set-name",
		k8sconsts.WorkloadNamespaceLabel: "already-set-namespace",
		k8sconsts.WorkloadKindLabel:      "already-set-kind",
		"odigos.io/data-stream-payments": "true",
	}

	require.NoError(t, defaulter.Default(context.Background(), source))

	assert.Equal(t, "already-set-name", source.Labels[k8sconsts.WorkloadNameLabel])
	assert.Equal(t, "already-set-namespace", source.Labels[k8sconsts.WorkloadNamespaceLabel])
	assert.Equal(t, "already-set-kind", source.Labels[k8sconsts.WorkloadKindLabel])
	// An existing data stream label means the Source is already grouped, so the default
	// data stream must not be added on top of it.
	assert.NotContains(t, source.Labels, defaultDataStreamLabel)
}

func TestSourcesDefaulter_AddsTheDefaultDataStreamWhenTheExistingLabelIsNotEnabled(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Labels = map[string]string{"odigos.io/data-stream-payments": "false"}

	require.NoError(t, defaulter.Default(context.Background(), source))

	assert.Equal(t, "true", source.Labels[defaultDataStreamLabel])
}

func TestSourcesDefaulter_ReplacesTheDeprecatedSplitFinalizers(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Finalizers = []string{
		k8sconsts.StartLangDetectionFinalizer,
		k8sconsts.DeleteInstrumentationConfigFinalizer,
		"third-party/finalizer",
	}

	require.NoError(t, defaulter.Default(context.Background(), source))

	assert.Equal(t, []string{"third-party/finalizer", k8sconsts.SourceInstrumentationFinalizer}, source.Finalizers)
}

func TestSourcesDefaulter_DoesNotDuplicateTheInstrumentationFinalizer(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Finalizers = []string{k8sconsts.SourceInstrumentationFinalizer}

	require.NoError(t, defaulter.Default(context.Background(), source))

	assert.Equal(t, []string{k8sconsts.SourceInstrumentationFinalizer}, source.Finalizers)
}

func TestSourcesDefaulter_DoesNotReAddTheFinalizerToADeletedSource(t *testing.T) {
	defaulter := &SourcesDefaulter{Client: newSourceTestClient(t)}
	source := newValidSource()
	deletedAt := metav1.Now()
	source.DeletionTimestamp = &deletedAt
	source.Finalizers = []string{k8sconsts.StartLangDetectionFinalizer}

	require.NoError(t, defaulter.Default(context.Background(), source))

	// Re-adding the finalizer while the Source is terminating would block the deletion.
	assert.Empty(t, source.Finalizers)
}

// ****************
// doesSourceHaveDataStreamLabel
// ****************

func TestDoesSourceHaveDataStreamLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   bool
	}{
		{name: "no labels", labels: nil, want: false},
		{name: "no data stream label", labels: map[string]string{k8sconsts.WorkloadNameLabel: sourceTestWorkload}, want: false},
		{name: "default data stream", labels: map[string]string{defaultDataStreamLabel: "true"}, want: true},
		{name: "custom data stream", labels: map[string]string{"odigos.io/data-stream-payments": "true"}, want: true},
		{name: "data stream label turned off", labels: map[string]string{"odigos.io/data-stream-payments": "false"}, want: false},
		{
			name:   "one of several data streams is enabled",
			labels: map[string]string{"odigos.io/data-stream-payments": "false", "odigos.io/data-stream-checkout": "true"},
			want:   true,
		},
		{
			name:   "a label that merely contains the prefix does not count",
			labels: map[string]string{"team.odigos.io/data-stream-payments": "true"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := newValidSource()
			source.Labels = tt.labels
			assert.Equal(t, tt.want, doesSourceHaveDataStreamLabel(source))
		})
	}
}

func TestDefaultDataStreamLabelMatchesTheDefaultDataStreamName(t *testing.T) {
	// The label the defaulter writes has to be the one the data stream machinery reads.
	assert.Equal(t, "odigos.io/data-stream-default", defaultDataStreamLabel)
	assert.Equal(t, "default", consts.DefaultDataStream)
}

// ****************
// SourcesValidator.ValidateCreate / validateSourceFields
// ****************

func TestSourcesValidator_ValidateCreateAcceptsAValidSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t)}

	warnings, err := validator.ValidateCreate(context.Background(), newValidSource())

	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

func TestSourcesValidator_ValidateCreateAcceptsAValidNamespaceSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t)}

	warnings, err := validator.ValidateCreate(context.Background(), newValidNamespaceSource())

	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

func TestSourcesValidator_ValidateCreateAcceptsAValidRegexSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t)}

	// A regex Source carries a hash instead of the workload name, so the
	// "label must match spec.workload.name" rule must not apply to it.
	warnings, err := validator.ValidateCreate(context.Background(), newRegexSource("checkout-regex-source", "checkout-.*"))

	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

func TestSourcesValidator_ValidateCreateAcceptsASingleDeprecatedFinalizer(t *testing.T) {
	// Only the combination of both deprecated finalizers is rejected; either one on its own
	// is what an upgraded Source looks like before the defaulter has replaced it.
	for _, finalizer := range []string{
		k8sconsts.StartLangDetectionFinalizer,
		k8sconsts.DeleteInstrumentationConfigFinalizer,
	} {
		t.Run(finalizer, func(t *testing.T) {
			validator := &SourcesValidator{Client: newSourceTestClient(t)}
			source := newValidSource()
			source.Finalizers = []string{finalizer}

			_, err := validator.ValidateCreate(context.Background(), source)

			assert.NoError(t, err)
		})
	}
}

func TestSourcesValidator_ValidateCreateRejectsInvalidSources(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*v1alpha1.Source)
		want   sourceValidationError
	}{
		{
			name: "both deprecated finalizers",
			mutate: func(s *v1alpha1.Source) {
				s.Finalizers = []string{k8sconsts.StartLangDetectionFinalizer, k8sconsts.DeleteInstrumentationConfigFinalizer}
			},
			want: sourceValidationError{"metadata.finalizers", "Source may only have one finalizer"},
		},
		{
			name: "workload-name label does not match the workload name",
			mutate: func(s *v1alpha1.Source) {
				s.Labels[k8sconsts.WorkloadNameLabel] = "not-checkout"
			},
			want: sourceValidationError{"metadata.labels", "odigos.io/workload-name must match spec.workload.name"},
		},
		{
			name: "workload-name label missing entirely",
			mutate: func(s *v1alpha1.Source) {
				delete(s.Labels, k8sconsts.WorkloadNameLabel)
			},
			want: sourceValidationError{"metadata.labels", "odigos.io/workload-name must match spec.workload.name"},
		},
		{
			name: "invalid regex pattern",
			mutate: func(s *v1alpha1.Source) {
				s.Spec.MatchWorkloadNameAsRegex = true
				s.Spec.Workload.Name = "checkout-(["
				s.Labels[k8sconsts.WorkloadNameLabel] = sourceTestNameHash
			},
			want: sourceValidationError{"spec.workload.name", "invalid regex pattern: error parsing regexp: missing closing ]: `[`"},
		},
		{
			name: "workload-namespace label does not match the workload namespace",
			mutate: func(s *v1alpha1.Source) {
				s.Labels[k8sconsts.WorkloadNamespaceLabel] = "other-ns"
			},
			want: sourceValidationError{"metadata.labels", "odigos.io/workload-namespace must match spec.workload.namespace"},
		},
		{
			name: "workload-kind label does not match the workload kind",
			mutate: func(s *v1alpha1.Source) {
				s.Labels[k8sconsts.WorkloadKindLabel] = string(k8sconsts.WorkloadKindStatefulSet)
			},
			want: sourceValidationError{"metadata.labels", "odigos.io/workload-kind must match spec.workload.kind"},
		},
		{
			name: "no data stream label",
			mutate: func(s *v1alpha1.Source) {
				delete(s.Labels, defaultDataStreamLabel)
			},
			want: sourceValidationError{"metadata.labels", "Source must have at least one odigos.io/data-stream-* label to indicate a data stream group"},
		},
		{
			name: "unsupported workload kind",
			mutate: func(s *v1alpha1.Source) {
				s.Spec.Workload.Kind = "ReplicaSet"
				s.Labels[k8sconsts.WorkloadKindLabel] = "ReplicaSet"
			},
			want: sourceValidationError{"spec.workload.kind", "workload kind must be one of (Deployment, DaemonSet, StatefulSet, Namespace, Rollout)"},
		},
		{
			name: "Source namespace differs from the workload namespace",
			mutate: func(s *v1alpha1.Source) {
				s.Namespace = "odigos-system"
			},
			want: sourceValidationError{"spec.workload.namespace", "Source namespace must match spec.workload.namespace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &SourcesValidator{Client: newSourceTestClient(t)}
			source := newValidSource()
			tt.mutate(source)

			warnings, err := validator.ValidateCreate(context.Background(), source)

			assert.Nil(t, warnings)
			require.Error(t, err)
			assertSourceValidationErrors(t, err, tt.want)
		})
	}
}

func TestSourcesValidator_ValidateCreateRejectsInvalidNamespaceSources(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*v1alpha1.Source)
		want   sourceValidationError
	}{
		{
			name: "workload name is not the namespace",
			mutate: func(s *v1alpha1.Source) {
				s.Spec.Workload.Name = sourceTestWorkload
				s.Labels[k8sconsts.WorkloadNameLabel] = sourceTestWorkload
			},
			want: sourceValidationError{"spec.workload.namespace", "namespace Source must have matching workload.name and workload.namespace"},
		},
		{
			name:   "otel service name is set",
			mutate: func(s *v1alpha1.Source) { s.Spec.OtelServiceName = "checkout-service" },
			want:   sourceValidationError{"spec.otelServiceName", "Service name is not valid for Namespace sources, only valid for Workload Sources"},
		},
		{
			name:   "workload name is matched as a regex",
			mutate: func(s *v1alpha1.Source) { s.Spec.MatchWorkloadNameAsRegex = true },
			want:   sourceValidationError{"spec.MatchWorkloadNameAsRegex", "MatchWorkloadNameAsRegex is not valid for Namespace sources, only valid for Workload Sources"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &SourcesValidator{Client: newSourceTestClient(t)}
			source := newValidNamespaceSource()
			tt.mutate(source)

			_, err := validator.ValidateCreate(context.Background(), source)

			require.Error(t, err)
			assertSourceValidationErrors(t, err, tt.want)
		})
	}
}

func TestSourcesValidator_ValidateCreateReportsEveryBrokenRule(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Labels[k8sconsts.WorkloadNameLabel] = "not-checkout"
	source.Labels[k8sconsts.WorkloadNamespaceLabel] = "other-ns"

	_, err := validator.ValidateCreate(context.Background(), source)

	require.Error(t, err)
	assertSourceValidationErrors(t, err,
		sourceValidationError{"metadata.labels", "odigos.io/workload-name must match spec.workload.name"},
		sourceValidationError{"metadata.labels", "odigos.io/workload-namespace must match spec.workload.namespace"},
	)
}

func TestSourcesValidator_ValidateCreateNamesTheSourceInTheError(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Labels[k8sconsts.WorkloadNameLabel] = "not-checkout"

	_, err := validator.ValidateCreate(context.Background(), source)

	statusErr := &apierrors.StatusError{}
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, "checkout-deployment-source", statusErr.ErrStatus.Details.Name)
	assert.Equal(t, "Source", statusErr.ErrStatus.Details.Kind)
	assert.Equal(t, "odigos.io", statusErr.ErrStatus.Details.Group)
}

func TestSourcesValidator_ValidateDeleteAlwaysAccepts(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t)}
	source := newValidSource()
	source.Labels = nil
	source.Spec.Workload.Kind = "ReplicaSet"

	warnings, err := validator.ValidateDelete(context.Background(), source)

	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

// ****************
// SourcesValidator.ValidateUpdate
// ****************

func TestSourcesValidator_ValidateUpdateAcceptsAnUnchangedSource(t *testing.T) {
	old := newValidSource()
	validator := &SourcesValidator{Client: newSourceTestClient(t, old.DeepCopy())}

	warnings, err := validator.ValidateUpdate(context.Background(), old, newValidSource())

	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

func TestSourcesValidator_ValidateUpdateAcceptsAMutableSpecChange(t *testing.T) {
	old := newValidSource()
	updated := newValidSource()
	updated.Spec.DisableInstrumentation = true
	updated.Spec.OtelServiceName = "checkout-service"
	validator := &SourcesValidator{Client: newSourceTestClient(t, old.DeepCopy())}

	warnings, err := validator.ValidateUpdate(context.Background(), old, updated)

	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

func TestSourcesValidator_ValidateUpdateRejectsImmutableChanges(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*v1alpha1.Source)
		want   []sourceValidationError
	}{
		{
			name:   "name",
			mutate: func(s *v1alpha1.Source) { s.Name = "renamed-source" },
			want: []sourceValidationError{
				{"metadata.name", "Source name is immutable"},
				// The uniqueness check skips the Source by name, so a rename also makes
				// the stored revision of the same Source look like a duplicate.
				{"spec.workload", "duplicate source(s) exist for workload: checkout-deployment-source"},
			},
		},
		{
			name:   "namespace",
			mutate: func(s *v1alpha1.Source) { s.Namespace = "other-ns" },
			want: []sourceValidationError{
				{"metadata.namespace", "Source namespace is immutable"},
				{"spec.workload.namespace", "Source namespace must match spec.workload.namespace"},
			},
		},
		{
			name: "workload-kind label",
			mutate: func(s *v1alpha1.Source) {
				s.Labels[k8sconsts.WorkloadKindLabel] = string(k8sconsts.WorkloadKindStatefulSet)
			},
			want: []sourceValidationError{
				{"metadata.labels", "Source workload-kind label is immutable"},
				{"metadata.labels", "odigos.io/workload-kind must match spec.workload.kind"},
			},
		},
		{
			name: "workload-name label",
			mutate: func(s *v1alpha1.Source) {
				s.Labels[k8sconsts.WorkloadNameLabel] = "not-checkout"
			},
			want: []sourceValidationError{
				{"metadata.labels", "Source workload-name label is immutable"},
				{"metadata.labels", "odigos.io/workload-name must match spec.workload.name"},
			},
		},
		{
			name:   "MatchWorkloadNameAsRegex",
			mutate: func(s *v1alpha1.Source) { s.Spec.MatchWorkloadNameAsRegex = true },
			want:   []sourceValidationError{{"spec.MatchWorkloadNameAsRegex", "Source MatchWorkloadNameAsRegex is immutable"}},
		},
		{
			name: "workload-namespace label",
			mutate: func(s *v1alpha1.Source) {
				s.Labels[k8sconsts.WorkloadNamespaceLabel] = "other-ns"
			},
			want: []sourceValidationError{
				{"metadata.labels", "Source workload-namespace label is immutable"},
				{"metadata.labels", "odigos.io/workload-namespace must match spec.workload.namespace"},
			},
		},
		{
			name:   "workload",
			mutate: func(s *v1alpha1.Source) { s.Spec.Workload.Kind = k8sconsts.WorkloadKindStatefulSet },
			want: []sourceValidationError{
				{"spec.workload", "Source workload is immutable"},
				{"metadata.labels", "odigos.io/workload-kind must match spec.workload.kind"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := newValidSource()
			updated := newValidSource()
			tt.mutate(updated)
			validator := &SourcesValidator{Client: newSourceTestClient(t, old.DeepCopy())}

			warnings, err := validator.ValidateUpdate(context.Background(), old, updated)

			assert.Nil(t, warnings)
			require.Error(t, err)
			assertSourceValidationErrors(t, err, tt.want...)
		})
	}
}

func TestSourcesValidator_ValidateUpdateNamesTheUpdatedSourceInTheError(t *testing.T) {
	old := newValidSource()
	updated := newValidSource()
	updated.Name = "renamed-source"
	validator := &SourcesValidator{Client: newSourceTestClient(t, old.DeepCopy())}

	_, err := validator.ValidateUpdate(context.Background(), old, updated)

	statusErr := &apierrors.StatusError{}
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, "renamed-source", statusErr.ErrStatus.Details.Name)
	assert.Equal(t, "Source", statusErr.ErrStatus.Details.Kind)
	assert.Equal(t, "odigos.io", statusErr.ErrStatus.Details.Group)
}

// ****************
// SourcesValidator.validateSourceUniqueness
// ****************

func TestSourcesValidator_RejectsADuplicateSourceForTheSameWorkload(t *testing.T) {
	existing := newValidSource()
	existing.Name = "checkout-deployment-source-copy"
	validator := &SourcesValidator{Client: newSourceTestClient(t, existing)}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	require.Error(t, err)
	assertSourceValidationErrors(t, err,
		sourceValidationError{"spec.workload", "duplicate source(s) exist for workload: checkout-deployment-source-copy"})
}

func TestSourcesValidator_ReportsEveryDuplicateSource(t *testing.T) {
	first := newValidSource()
	first.Name = "checkout-source-a"
	second := newValidSource()
	second.Name = "checkout-source-b"
	validator := &SourcesValidator{Client: newSourceTestClient(t, first, second)}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	require.Error(t, err)
	assertSourceValidationErrors(t, err,
		sourceValidationError{"spec.workload", "duplicate source(s) exist for workload: checkout-source-a,checkout-source-b"})
}

func TestSourcesValidator_UpdateDoesNotTreatTheSourceItselfAsADuplicate(t *testing.T) {
	old := newValidSource()
	updated := newValidSource()
	updated.Spec.OtelServiceName = "checkout-service"
	validator := &SourcesValidator{Client: newSourceTestClient(t, old.DeepCopy())}

	_, err := validator.ValidateUpdate(context.Background(), old, updated)

	assert.NoError(t, err)
}

func TestSourcesValidator_AllowsASourceForADifferentWorkloadInTheSameNamespace(t *testing.T) {
	existing := newValidSource()
	existing.Name = "frontend-deployment-source"
	existing.Labels[k8sconsts.WorkloadNameLabel] = "frontend"
	existing.Spec.Workload.Name = "frontend"
	validator := &SourcesValidator{Client: newSourceTestClient(t, existing)}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	assert.NoError(t, err)
}

func TestSourcesValidator_UniquenessIsScopedToTheWorkloadKindAndNamespace(t *testing.T) {
	// Sources for the same workload name but a different kind, or in a different
	// namespace, are distinct and must not be reported as duplicates. The label selector
	// in validateSourceUniqueness is what keeps them apart.
	sameNameOtherKind := newValidSource()
	sameNameOtherKind.Name = "checkout-statefulset-source"
	sameNameOtherKind.Labels[k8sconsts.WorkloadKindLabel] = string(k8sconsts.WorkloadKindStatefulSet)
	sameNameOtherKind.Spec.Workload.Kind = k8sconsts.WorkloadKindStatefulSet

	sameNameOtherNamespace := newValidSource()
	sameNameOtherNamespace.Namespace = "other-ns"
	sameNameOtherNamespace.Labels[k8sconsts.WorkloadNamespaceLabel] = "other-ns"
	sameNameOtherNamespace.Spec.Workload.Namespace = "other-ns"

	validator := &SourcesValidator{Client: newSourceTestClient(t, sameNameOtherKind, sameNameOtherNamespace)}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	assert.NoError(t, err)
}

func TestSourcesValidator_UniquenessFiltersOnBothTheSourceNamespaceAndTheWorkloadNamespaceLabel(t *testing.T) {
	// The duplicate lookup restricts by the Source's own namespace and by the
	// workload-namespace label independently. A stored Source whose two namespaces
	// disagree - which the "Source namespace must match spec.workload.namespace" rule
	// prevents today, but which older Sources can still be in - must not be reported as a
	// duplicate of a valid Source.
	mislabelled := newValidSource()
	mislabelled.Name = "checkout-mislabelled-source"
	mislabelled.Labels[k8sconsts.WorkloadNamespaceLabel] = "other-ns"

	elsewhere := newValidSource()
	elsewhere.Name = "checkout-source-elsewhere"
	elsewhere.Namespace = "other-ns"

	validator := &SourcesValidator{Client: newSourceTestClient(t, mislabelled, elsewhere)}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	assert.NoError(t, err)
}

func TestSourcesValidator_RejectsARegexSourceThatMatchesAnExistingExactSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t, newValidSource())}

	_, err := validator.ValidateCreate(context.Background(), newRegexSource("checkout-regex-source", "check.*"))

	require.Error(t, err)
	assertSourceValidationErrors(t, err,
		sourceValidationError{"spec.workload", "duplicate source(s) exist for workload: checkout-deployment-source"})
}

func TestSourcesValidator_AllowsARegexSourceThatMatchesNoExistingSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t, newValidSource())}

	_, err := validator.ValidateCreate(context.Background(), newRegexSource("frontend-regex-source", "frontend-.*"))

	assert.NoError(t, err)
}

func TestSourcesValidator_RejectsAnExactSourceCoveredByAnExistingRegexSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t, newRegexSource("checkout-regex-source", "check.*"))}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	require.Error(t, err)
	assertSourceValidationErrors(t, err,
		sourceValidationError{"spec.workload", "duplicate source(s) exist for workload: checkout-regex-source"})
}

func TestSourcesValidator_AllowsAnExactSourceNotCoveredByAnExistingRegexSource(t *testing.T) {
	validator := &SourcesValidator{Client: newSourceTestClient(t, newRegexSource("frontend-regex-source", "frontend-.*"))}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	assert.NoError(t, err)
}

func TestSourcesValidator_AllowsTwoOverlappingRegexSources(t *testing.T) {
	// Two regex Sources are always allowed, even when their patterns overlap, because the
	// set of workloads each one matches is not known at admission time.
	validator := &SourcesValidator{Client: newSourceTestClient(t, newRegexSource("checkout-regex-source", "check.*"))}

	_, err := validator.ValidateCreate(context.Background(), newRegexSource("checkout-regex-source-2", "checkout.*"))

	assert.NoError(t, err)
}

func TestSourcesValidator_AnUncompilableExistingRegexDoesNotBlockAnExactSource(t *testing.T) {
	// An already-stored Source can hold a pattern that does not compile (it may predate
	// the validation, or have been written straight to the API server); it must not fail
	// the admission of an unrelated Source.
	validator := &SourcesValidator{Client: newSourceTestClient(t, newRegexSource("broken-regex-source", "check(["))}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	assert.NoError(t, err)
}

func TestSourcesValidator_SurfacesAFailureToListExistingSources(t *testing.T) {
	// A failed lookup must not be mistaken for "no duplicates exist".
	failingClient := fake.NewClientBuilder().
		WithScheme(sourceTestScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return errors.New("etcd is unavailable")
			},
		}).
		Build()
	validator := &SourcesValidator{Client: failingClient}

	_, err := validator.ValidateCreate(context.Background(), newValidSource())

	require.Error(t, err)
	assertSourceValidationErrors(t, err, sourceValidationError{"spec.workload", "etcd is unavailable"})
}

// ****************
// Defaulter and validator agree
// ****************

func TestSourcesDefaulterProducesASourceTheValidatorAccepts(t *testing.T) {
	// The defaulter runs before the validator in the admission chain, so anything it
	// produces from a minimal Source has to pass validation.
	tests := []struct {
		name     string
		workload k8sconsts.PodWorkload
	}{
		{
			name:     "deployment",
			workload: k8sconsts.PodWorkload{Name: sourceTestWorkload, Namespace: sourceTestNamespace, Kind: k8sconsts.WorkloadKindDeployment},
		},
		{
			name:     "namespace",
			workload: k8sconsts.PodWorkload{Name: sourceTestNamespace, Namespace: sourceTestNamespace, Kind: k8sconsts.WorkloadKindNamespace},
		},
		{
			name:     "cron job",
			workload: k8sconsts.PodWorkload{Name: sourceTestWorkload, Namespace: sourceTestNamespace, Kind: k8sconsts.WorkloadKindCronJob},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := newSourceTestClient(t)
			source := &v1alpha1.Source{
				ObjectMeta: metav1.ObjectMeta{Name: "generated-source", Namespace: sourceTestNamespace},
				Spec:       v1alpha1.SourceSpec{Workload: tt.workload},
			}

			require.NoError(t, (&SourcesDefaulter{Client: k8sClient}).Default(context.Background(), source))

			_, err := (&SourcesValidator{Client: k8sClient}).ValidateCreate(context.Background(), source)
			assert.NoError(t, err)
			assert.True(t, controllerutil.ContainsFinalizer(source, k8sconsts.SourceInstrumentationFinalizer))
		})
	}
}
